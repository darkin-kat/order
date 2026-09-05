package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	mongorepo "github.com/darkin-kat/order/internal/repository/mongo"
	"github.com/darkin-kat/order/internal/server"
	ordersv1 "github.com/darkin-kat/store-api/gen/orders/v1"
	productsv1 "github.com/darkin-kat/store-api/gen/products/v1"
	usrv1 "github.com/darkin-kat/store-api/gen/users/v1"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type Config struct {
	MongoURI       string
	MongoDbName    string
	GRPCPort       string
	UserService    string
	ProductService string
}

func loadConfig() Config {
	return Config{
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDbName:    getEnv("MONGO_DB_NAME", "project_two"),
		GRPCPort:       getEnv("GRPC_PORT", "50053"),
		UserService:    getEnv("USER_SERVICE_ADDR", "localhost:50051"),
		ProductService: getEnv("PRODUCT_SERVICE_ADDR", "localhost:50052"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func connectToMongo(ctx context.Context, uri string) (*mongo.Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err = client.Ping(connectCtx, nil); err != nil {
		return nil, err
	}
	return client, nil
}

func main() {
	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoClient, err := connectToMongo(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.MongoDbName)
	repo := mongorepo.NewMongoRepository(db)

	usersConn, err := grpc.NewClient(
		cfg.UserService,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to users-service: %v", err)
	}
	defer usersConn.Close()

	usersClient := usrv1.NewUserServiceClient(usersConn)

	productConn, err := grpc.NewClient(
		cfg.ProductService,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to product-service: %v", err)
	}
	defer productConn.Close()

	productClient := productsv1.NewProductsServiceClient(productConn)

	srv := server.NewServer(repo, repo, usersClient, productClient)

	grpcServer := grpc.NewServer()
	ordersv1.RegisterOrderServiceServer(grpcServer, srv)
	ordersv1.RegisterCartServiceServer(grpcServer, srv)
	reflection.Register(grpcServer)

	// TODO: защититься от паник вообще, через recovery interceptor

	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	go func() {
		log.Printf("gRPC order service is listening on port %s", cfg.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received, stopping gracefully...")
	grpcServer.GracefulStop()
	log.Println("gRPC server stopped")
}
