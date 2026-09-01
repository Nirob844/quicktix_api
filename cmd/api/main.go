package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quicktix/internal/auth"
	"quicktix/internal/platform/database"
	"quicktix/internal/platform/middleware"
	pkgredis "quicktix/internal/platform/redis"
	"quicktix/internal/platform/router"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quicktix:quicktix@localhost:5433/quicktix?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6380"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-jwt-key-quicktix"
	}

	db, err := database.Connect(database.Config{
		DSN: dbURL,
	})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close database connection", "error", err)
		} else {
			slog.Info("database connection closed")
		}
	}()
	slog.Info("database connection established successfully")

	rdb, err := pkgredis.Connect(pkgredis.Config{
		Addr: redisAddr,
	})
	if err != nil {
		slog.Warn("failed to connect to redis on startup", "error", err, "addr", redisAddr)
	} else {
		defer func() {
			if err := rdb.Close(); err != nil {
				slog.Error("failed to close redis client", "error", err)
			} else {
				slog.Info("redis client connection closed")
			}
		}()
		slog.Info("redis connection established successfully")
	}

	if os.Getenv("DB_AUTO_MIGRATE") != "false" {
		if err := database.RunMigrationsPath(db.DB, "migrations"); err != nil {
			slog.Error("failed to run database migrations", "error", err)
			os.Exit(1)
		}
	}

	// Initialize Auth module
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, rdb, jwtSecret, 24*time.Hour)
	authHandler := auth.NewHandler(authService)

	r := router.New()
	r.Use(middleware.Recovery) // outermost — catches panics from everything inside
	r.Use(middleware.Logging)

	// Base system routes
	r.Handle("GET /ping", http.HandlerFunc(handlePing))
	r.Handle("GET /healthz", handleHealthz(db, rdb))
	r.Handle("GET /events/{id}", http.HandlerFunc(handleGetEvent)) // path param demo

	// Register Domain Module Routes
	authHandler.RegisterRoutes(r, jwtSecret)

	// RBAC Protected Demo Endpoints
	r.Handle("GET /api/v1/organizer/dashboard", middleware.Authenticate(jwtSecret)(middleware.RequireOrganizer()(http.HandlerFunc(handleOrganizerDashboard))))
	r.Handle("GET /api/v1/admin/dashboard", middleware.Authenticate(jwtSecret)(middleware.RequireAdmin()(http.HandlerFunc(handleAdminDashboard))))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r.Chain(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutdown signal received, starting graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped cleanly")
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"pong"}`))
}

func handleHealthz(db *sqlx.DB, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "connected"
		redisStatus := "connected"
		healthy := true

		if err := db.PingContext(ctx); err != nil {
			slog.Error("health check failed: database unreachable", "error", err)
			dbStatus = "unreachable"
			healthy = false
		}

		if rdb == nil {
			redisStatus = "unreachable"
			healthy = false
		} else if err := rdb.Ping(ctx).Err(); err != nil {
			slog.Error("health check failed: redis unreachable", "error", err)
			redisStatus = "unreachable"
			healthy = false
		}

		if !healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","database":"` + dbStatus + `","redis":"` + redisStatus + `"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","database":"` + dbStatus + `","redis":"` + redisStatus + `"}`))
	}
}

func handleGetEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Go 1.22 built-in path param extraction
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"event_id":"` + id + `"}`))
}

func handleOrganizerDashboard(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUserFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "welcome organizer",
		"user_id": claims.UserID,
		"role":    claims.Role,
	})
}

func handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUserFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "welcome admin",
		"user_id": claims.UserID,
		"role":    claims.Role,
	})
}
