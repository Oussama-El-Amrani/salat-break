package main

import (
"io"
"log"
"os"
"os/exec"
"time"

"github.com/spf13/cobra"
"github.com/spf13/viper"

"github.com/Oussama-El-Amrani/salat-break/internal/cache"
"github.com/Oussama-El-Amrani/salat-break/internal/checker"
"github.com/Oussama-El-Amrani/salat-break/internal/location"
"github.com/Oussama-El-Amrani/salat-break/internal/media"
"github.com/Oussama-El-Amrani/salat-break/internal/notification"
"github.com/Oussama-El-Amrani/salat-break/internal/prayer"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "salat-break",
	Short:   "Automatically pause media players during prayer times",
	Version: Version,
	Run:     runRoot,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update salat-break to the latest version",
	Run:   runUpdate,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.Flags().Int("notification-timeout", 10000, "Timeout for notifications in milliseconds (hides the popup)")
	rootCmd.Flags().Int("notification-clear-delay", 300000, "Delay in milliseconds before clearing notification from the system tray (default 5m)")
	rootCmd.Flags().Bool("test-pause", false, "Run test: pause all media players")
	rootCmd.Flags().Bool("test-play", false, "Run test: resume all media players")
	rootCmd.Flags().Bool("test-notify", false, "Run test: send a test notification")
	rootCmd.Flags().Bool("test-logic", false, "Run simulation to test prayer window logic")
	rootCmd.Flags().Bool("show-timings", false, "Display today's prayer times for your current location and exit")
	rootCmd.Flags().String("city", "", "Manually override the auto-detected city (sets a persistent preference)")
	rootCmd.Flags().String("country", "", "Manually override the auto-detected country (sets a persistent preference)")
	rootCmd.Flags().Int("method", 0, "Set the prayer calculation method ID (e.g., 21 for Morocco, 3 for MWL, 0 for auto-detection)")

	_ = viper.BindPFlags(rootCmd.Flags())
	viper.SetEnvPrefix("SALAT_BREAK")
	viper.AutomaticEnv()

	rootCmd.AddCommand(updateCmd)
}

func runRoot(cmd *cobra.Command, args []string) {
	// Initialize Services
	timeout := viper.GetInt("notification-timeout")
	clearDelay := viper.GetInt("notification-clear-delay")

	notifier := notification.NewService(timeout, clearDelay)
	mediaCtrl := media.NewController(notifier)
	locationSvc := location.NewService()
	prayerSvc := prayer.NewService()
	checkSvc := checker.NewService(mediaCtrl, notifier)

	// Handle Tests
	if viper.GetBool("test-pause") {
		log.Println("Test: Pausing all media players...")
		mediaCtrl.PauseAllPlayers()
		return
	}
	if viper.GetBool("test-play") {
		log.Println("Test: Resuming all media players...")
		mediaCtrl.PlayAllPlayers()
		return
	}
	if viper.GetBool("test-notify") {
		log.Println("Test: Sending notification...")
		notifier.SendNotification("Salat Break Test", "This is a test notification for the Salat Break app.")
		time.Sleep(time.Duration(clearDelay+1000) * time.Millisecond)
		return
	}
	if viper.GetBool("test-logic") {
		runSimulation(checkSvc)
		return
	}
	if viper.GetBool("show-timings") {
		loc, err := locationSvc.GetLocation()
		if err != nil {
			log.Fatalf("Error getting location: %v", err)
		}
		prayerSvc.Method = loc.Method
		_, err = prayerSvc.GetPrayerTimes(loc)
		if err != nil {
			log.Fatalf("Error getting prayer times: %v", err)
		}
		// Timings are logged by GetPrayerTimes
		return
	}

	// Handle Location Overrides
	city := viper.GetString("city")
	country := viper.GetString("country")
	method := viper.GetInt("method")
	if city != "" || country != "" || method > 0 {
		var override location.Location
		_ = cache.Load("location_override.json", &override)
		if city != "" {
			override.City = city
		}
		if country != "" {
			override.Country = country
		}
		if method > 0 {
			override.Method = method
		}
		if err := cache.Save("location_override.json", override); err != nil {
			log.Fatalf("Error saving location override: %v", err)
		}
		log.Printf("Location configuration saved: City=%s, Country=%s, Method=%d", override.City, override.Country, override.Method)

		// Fetch and log timings for confirmation
		loc, err := locationSvc.GetLocation()
		if err == nil {
			_, _ = prayerSvc.GetPrayerTimes(loc)
		}

		// Restart systemd service to apply changes
		log.Println("Restarting salat-break service to apply changes...")
		_ = exec.Command("systemctl", "--user", "restart", "salat-break.service").Run()
		return
	}

	// Main Loop
	log.Println("Salat Break service started. Monitoring prayer times...")
	for {
		loc, err := locationSvc.GetLocation()
		if err != nil {
			log.Printf("Error getting location: %v. Retrying in 1 minute...", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		prayerSvc.Method = loc.Method
		timings, err := prayerSvc.GetPrayerTimes(loc)
		if err != nil {
			log.Printf("Error getting prayer times: %v. Retrying in 1 minute...", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		checkSvc.CheckAndPause(timings.Data.Timings)
		time.Sleep(30 * time.Second)
	}
}

func runSimulation(checkSvc *checker.Service) {
	log.Println("--- Starting Logic Simulation ---")
	testTimings := map[string]string{"Dhuhr": "12:00"}
	simTime := time.Date(2026, 3, 31, 11, 56, 0, 0, time.Local)
	for i := 0; i < 10; i++ {
		log.Printf("[Sim] Time: %s", simTime.Format("15:04:05"))
		checkSvc.CheckAndPause(testTimings, simTime)
		simTime = simTime.Add(1 * time.Minute)
	}
	log.Println("--- Simulation Complete ---")
}

func runUpdate(cmd *cobra.Command, args []string) {
	log.Println("Checking for updates...")
	installerURL := "https://raw.githubusercontent.com/Oussama-El-Amrani/salat-break/main/install.sh"
	
	// Create the command: curl -sSL <url> | bash
	c1 := exec.Command("curl", "-sSL", installerURL)
	c2 := exec.Command("bash")

	// Create a pipe between the two commands
	r, w := io.Pipe()
	c1.Stdout = w
	c2.Stdin = r
	c2.Stdout = os.Stdout
	c2.Stderr = os.Stderr

	// Start both commands
	if err := c1.Start(); err != nil {
		log.Fatalf("Error starting curl: %v", err)
	}
	if err := c2.Start(); err != nil {
		log.Fatalf("Error starting bash: %v", err)
	}

	// Wait for curl to finish and close the pipe
	go func() {
		_ = c1.Wait()
		_ = w.Close()
	}()

	// Wait for bash to finish
	if err := c2.Wait(); err != nil {
		log.Fatalf("Error during update: %v", err)
	}
	
	log.Println("Update process completed.")
}
