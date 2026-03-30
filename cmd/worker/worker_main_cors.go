port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("PORT_WORKER")
	}
	PORT := ":" + port
	if PORT == ":" {
		log.Fatal("PORT is not set in config.env")
	}