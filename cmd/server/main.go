package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xsm/internal/auth"
	"xsm/internal/config"
	"xsm/internal/database"
	"xsm/internal/encryption"
	"xsm/internal/handlers"
	ws "xsm/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("🚀 Запуск XSM - eXtreme Secure Messenger")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Ошибка загрузки конфигурации:", err)
	}

	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Ошибка миграций:", err)
	}

	// Инициализируем WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// Сервисы
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.Expiration)
	rsaService := encryption.NewRSAService(cfg.Security.RSAKeySize)
	authService := auth.NewAuthService(db.DB, jwtService, rsaService)

	// Обработчики
	authHandler := auth.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(db.DB)
	chatHandler := handlers.NewChatHandler(db.DB)
	messageHandler := handlers.NewMessageHandler(db.DB, hub)
	wsHandler := ws.NewWSHandler(hub, jwtService)

	router := gin.Default()

	// CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Статика
	router.Static("/web", "./web")
	router.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	// API
	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)

		protected := api.Group("/")
		protected.Use(auth.AuthMiddleware(jwtService))
		{
			// Профиль и пользователи
			protected.GET("/user/profile", authHandler.GetProfile)
			protected.GET("/users/search", userHandler.SearchUsers)
			protected.PUT("/user/profile", authHandler.UpdateProfile)
			protected.POST("/user/avatar", authHandler.UploadAvatar)

			// Чаты
			protected.POST("/chats", chatHandler.CreateChat)
			protected.GET("/chats", chatHandler.GetChats)
			protected.DELETE("/chats/:id", chatHandler.DeleteChat)

			// Сообщения
			protected.POST("/messages", messageHandler.SendMessage)
			protected.GET("/messages/:chat_id", messageHandler.GetMessages)
			protected.DELETE("/messages/:id", messageHandler.DeleteMessage)
			protected.POST("/messages/read", messageHandler.MarkAsRead)
			protected.GET("/messages/unread", messageHandler.GetUnreadCount)

			// WebSocket (без middleware)
			router.GET("/api/v1/ws", wsHandler.HandleWebSocket)
		}
	}

	srv := &http.Server{
		Addr:    cfg.Server.Host + ":" + cfg.Server.Port,
		Handler: router,
	}

	go func() {
		log.Printf("🌐 Сервер запущен на %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Ошибка сервера:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Завершение работы...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Принудительное завершение:", err)
	}

	log.Println("✅ Сервер остановлен")
}
