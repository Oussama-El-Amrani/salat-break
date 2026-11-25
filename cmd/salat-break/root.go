package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
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
	rootCmd.Flags().Bool("verbose", false, "Show detailed internal logs for location resolution and hardware scanning")
	rootCmd.Flags().Bool("test-pause", false, "Run test: pause all media players")
	rootCmd.Flags().Bool("test-play", false, "Run test: resume all media players")
	rootCmd.Flags().Bool("test-notify", false, "Run test: send a test notification")
	rootCmd.Flags().Bool("test-logic", false, "Run simulation to test prayer window logic")
	rootCmd.Flags().Bool("show-timings", false, "Display today's prayer times for your current location and exit")
	rootCmd.Flags().Bool("show-location", false, "Display your current calculated location and exit")
	rootCmd.Flags().String("city", "", "Manually override the auto-detected city (sets a persistent preference)")
	rootCmd.Flags().String("country", "", "Manually override the auto-detected country (sets a persistent preference)")
	rootCmd.Flags().Float64("lat", 0, "Manually set latitude for precise location (e.g., 33.5731 for Casablanca)")
	rootCmd.Flags().Float64("lon", 0, "Manually set longitude for precise location (e.g., -7.5898 for Casablanca)")
	rootCmd.Flags().Int("method", 0, "Set the prayer calculation method ID (e.g., 21 for Morocco, 3 for MWL, 0 for auto-detection)")
	rootCmd.Flags().Bool("reset-location", false, "Clear all manual location overrides and return to automatic detection")
	rootCmd.Flags().Bool("auto-play", true, "Automatically resume media playback after the prayer break ends")
	rootCmd.Flags().Bool("allow-playback", false, "Allow manual media playback to continue/resume during the prayer break")
	rootCmd.Flags().String("toggle-provider", "", "Toggle allowing a specific media provider (e.g., vlc, spotify, mpv) to play during Salat slots (sets a persistent preference)")
	rootCmd.Flags().String("allow-provider", "", "Allow specific media provider(s) (comma-separated, e.g., vlc,mpv) to play during Salat slots (sets a persistent preference)")
	rootCmd.Flags().String("disallow-provider", "", "Disallow specific media provider(s) from playing during Salat slots (sets a persistent preference)")
	rootCmd.Flags().Bool("list-providers", false, "List active media providers and show which providers are allowed during Salat slots")

	_ = viper.BindPFlags(rootCmd.Flags())
	viper.SetEnvPrefix("SALAT_BREAK")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(updateCmd)
}

func formatAllowedProviders(list []string) string {
	if len(list) == 0 {
		return "none"
	}
	return strings.Join(list, ", ")
}

func syncMediaConfig(mediaCtrl *media.Controller, cliExtra string) {
	cfg, _ := media.LoadConfig()
	merged := append([]string{}, cfg.AllowedProviders...)
	if cliExtra != "" {
		merged = append(merged, cliExtra)
	}
	mediaCtrl.SetAllowedProviders(merged)
}

func printProvidersList(mediaCtrl *media.Controller) {
	cfg, _ := media.LoadConfig()
	mediaCtrl.SetAllowedProviders(cfg.AllowedProviders)

	players := mediaCtrl.GetAllPlayers()
	fmt.Println("Active Media Providers:")
	if len(players) == 0 {
		fmt.Println("  (no active media players detected)")
	} else {
		for _, p := range players {
			meta := mediaCtrl.GetMetadata(p)
			prov := media.ResolveProviderName(p, meta)
			status := mediaCtrl.GetPlaybackStatus(p)
			if status == "" {
				status = "Unknown"
			}
			slotPolicy := "Paused during Salat"
			if mediaCtrl.IsProviderAllowed(p, meta) {
				slotPolicy = "Allowed during Salat"
			} else if !mediaCtrl.IsMusic(p, meta) {
				slotPolicy = "Ignored (non-music browser tab)"
			}
			trackInfo := ""
			if title := meta["title"]; title != "" {
				if artist := meta["artist"]; artist != "" {
					trackInfo = fmt.Sprintf(" [%s - %s]", artist, title)
				} else {
					trackInfo = fmt.Sprintf(" [%s]", title)
				}
			}
			fmt.Printf("  - %-16s (%s) | Status: %-8s | %s%s\n", prov, p, status, slotPolicy, trackInfo)
		}
	}
	fmt.Printf("\nAllowed Providers in Salat Slot: %s\n", formatAllowedProviders(cfg.AllowedProviders))
}

func runRoot(cmd *cobra.Command, args []string) {
	// Initialize Services
	timeout := viper.GetInt("notification-timeout")
	clearDelay := viper.GetInt("notification-clear-delay")

	notifier := notification.NewService(timeout, clearDelay)
	mediaCtrl := media.NewController(notifier)
	syncMediaConfig(mediaCtrl, viper.GetString("allow-provider"))

	locationSvc := location.NewService()
	locationSvc.Verbose = viper.GetBool("verbose")
	prayerSvc := prayer.NewService()
	if val := os.Getenv("SALAT_BREAK_ALADHAN_URL"); val != "" {
		prayerSvc.BaseURL = val
	}
	checkSvc := checker.NewService(mediaCtrl, notifier)
	checkSvc.AutoPlay = viper.GetBool("auto-play")
	checkSvc.AllowPlayback = viper.GetBool("allow-playback")

	// Handle Provider Listing
	if viper.GetBool("list-providers") {
		if !viper.GetBool("verbose") {
			log.SetOutput(io.Discard)
		}
		printProvidersList(mediaCtrl)
		return
	}

	// Handle Provider Toggle (CLI persistent toggle)
	if toggleProv := strings.TrimSpace(viper.GetString("toggle-provider")); toggleProv != "" {
		norm := media.NormalizeProviderName(toggleProv)
		if norm == "" {
			fmt.Fprintf(os.Stderr, "Error: invalid provider name %q\n", toggleProv)
			os.Exit(1)
		}
		allowed, updated, err := media.ToggleProvider(norm)
		if err != nil {
			log.Fatalf("Error saving media provider configuration: %v", err)
		}
		if allowed {
			fmt.Printf("Provider %q is now ALLOWED to play during Salat break slots.\n", norm)
		} else {
			fmt.Printf("Provider %q is now DISALLOWED (will be paused) during Salat break slots.\n", norm)
		}
		fmt.Printf("Allowed providers: %s\n", formatAllowedProviders(updated))
		restartService()
		return
	}

	// Handle Tests
	if viper.GetBool("test-pause") {
		log.Println("Test: Pausing all media players...")
		mediaCtrl.PauseAllPlayers()
		return
	}
	if viper.GetBool("test-play") {
		log.Println("Test: Resuming all media players...")
		mediaCtrl.ForcePlayAllPlayers()
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

	// Handle Explicit Allow / Disallow Provider Configuration
	allowProv := strings.TrimSpace(viper.GetString("allow-provider"))
	disallowProv := strings.TrimSpace(viper.GetString("disallow-provider"))
	if allowProv != "" || disallowProv != "" {
		var updated []string
		var err error
		if allowProv != "" {
			updated, err = media.AllowProviders(allowProv)
			if err != nil {
				log.Fatalf("Error saving allowed providers: %v", err)
			}
			fmt.Printf("Added allowed provider(s) for Salat break slots: %s\n", strings.Join(media.SplitProviderInput(allowProv), ", "))
		}
		if disallowProv != "" {
			updated, err = media.DisallowProviders(disallowProv)
			if err != nil {
				log.Fatalf("Error saving disallowed providers: %v", err)
			}
			fmt.Printf("Removed provider(s) from Salat break allowlist: %s\n", strings.Join(media.SplitProviderInput(disallowProv), ", "))
		}
		fmt.Printf("Allowed providers: %s\n", formatAllowedProviders(updated))
		restartService()
		return
	}

	if viper.GetBool("show-location") {
		if !viper.GetBool("verbose") {
			log.SetOutput(io.Discard)
		}
		loc, err := locationSvc.GetLocation()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting location: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Current Calculated Location:\n")
		fmt.Printf("  City:      %s\n", loc.City)
		fmt.Printf("  Country:   %s\n", loc.Country)
		fmt.Printf("  Latitude:  %.6f\n", loc.Lat)
		fmt.Printf("  Longitude: %.6f\n", loc.Lon)
		fmt.Printf("  Timezone:  %s\n", loc.Timezone)
		fmt.Printf("  Source:    %s\n", loc.Source)
		return
	}
	if viper.GetBool("show-timings") {
		if !viper.GetBool("verbose") {
			log.SetOutput(io.Discard)
		}
		loc, err := locationSvc.GetLocation()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting location: %v\n", err)
			os.Exit(1)
		}
		prayerSvc.Method = loc.Method
		timings, err := prayerSvc.GetPrayerTimes(loc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting prayer times: %v\n", err)
			os.Exit(1)
		}
		printTimingsUX(loc, prayerSvc, timings)
		return
	}

	// Handle Location Overrides
	city := viper.GetString("city")
	country := viper.GetString("country")
	lat := viper.GetFloat64("lat")
	lon := viper.GetFloat64("lon")
	method := viper.GetInt("method")
	if city != "" || country != "" || lat != 0 || lon != 0 || method > 0 {
		var override location.Location
		_ = cache.Load("location_override.json", &override)
		if city != "" {
			override.City = city
		}
		if country != "" {
			override.Country = country
		}
		if lat != 0 {
			override.Lat = lat
		}
		if lon != 0 {
			override.Lon = lon
		}
		if method > 0 {
			override.Method = method
		}
		if err := cache.Save("location_override.json", override); err != nil {
			log.Fatalf("Error saving location override: %v", err)
		}
		log.Printf("Location configuration saved: City=%s, Country=%s, Lat=%.4f, Lon=%.4f, Method=%d",
			override.City, override.Country, override.Lat, override.Lon, override.Method)

		// Fetch and log timings for confirmation
		loc, err := locationSvc.GetLocation()
		if err == nil {
			prayerSvc.Method = loc.Method
			_, _ = prayerSvc.GetPrayerTimes(loc)
		}

		// Restart service to apply changes
		restartService()
		return
	}

	if viper.GetBool("reset-location") {
		if err := cache.Delete("location_override.json"); err != nil {
			log.Printf("No manual location override to clear.")
		} else {
			log.Printf("Manual location overrides cleared. Returning to automatic detection.")
		}
		// Restart service to apply changes
		restartService()
		return
	}

	// Main Loop
	log.Println("Salat Break service started. Monitoring prayer times...")
	watcher := location.NewWatcher(locationSvc)
	var lastCity string

	for {
		loc, err := watcher.GetLocation()
		if err != nil {
			log.Printf("Error getting location: %v. Retrying in 1 minute...", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		// Detect significant location change (city changed)
		if lastCity != "" && lastCity != loc.City {
			log.Printf("Location change detected: %s -> %s. Updating prayer times.", lastCity, loc.City)
			notifier.SendNotification("Location Updated", fmt.Sprintf("Salat Break is now using prayer times for %s, %s.", loc.City, loc.Country))
		}
		lastCity = loc.City

		prayerSvc.Method = loc.Method
		timings, err := prayerSvc.GetPrayerTimes(loc)
		if err != nil {
			log.Printf("Error getting prayer times: %v. Retrying in 1 minute...", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		// Reload media provider allowlist dynamically on each check
		syncMediaConfig(mediaCtrl, "")
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
