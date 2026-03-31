package main

import (
"log"
"time"

"github.com/spf13/cobra"
"github.com/spf13/viper"

"github.com/oussama_ib0/salat-break/internal/checker"
"github.com/oussama_ib0/salat-break/internal/location"
"github.com/oussama_ib0/salat-break/internal/media"
"github.com/oussama_ib0/salat-break/internal/notification"
"github.com/oussama_ib0/salat-break/internal/prayer"
)

var rootCmd = &cobra.Command{
	Use:   "salat-break",
	Short: "Automatically pause media players during prayer times",
	Run:   runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.Flags().Int("notification-timeout", 10000, "Timeout for notifications in milliseconds (hides popup)")
	rootCmd.Flags().Int("notification-clear-delay", 300000, "Delay in milliseconds before clearing notification from tray (5 minutes)")
	rootCmd.Flags().Bool("test-pause", false, "Run test: pause all media players")
	rootCmd.Flags().Bool("test-play", false, "Run test: resume all media players")
	rootCmd.Flags().Bool("test-notify", false, "Run test: send a test notification")
	rootCmd.Flags().Bool("test-logic", false, "Run simulation to test prayer window logic")

	_ = viper.BindPFlags(rootCmd.Flags())
	viper.SetEnvPrefix("SALAT_BREAK")
	viper.AutomaticEnv()
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

	// Main Loop
	log.Println("Salat Break service started. Monitoring prayer times...")
	for {
		loc, err := locationSvc.GetLocation()
		if err != nil {
			log.Printf("Error getting location: %v. Retrying in 1 minute...", err)
			time.Sleep(1 * time.Minute)
			continue
		}

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
