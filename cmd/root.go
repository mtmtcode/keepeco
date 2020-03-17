package cmd

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/mitchellh/go-homedir"

	"github.com/atotto/clipboard"
	"github.com/l3msh0/go-fuzzyfinder"
	"github.com/l3msh0/keepeco/internal/cache"
	"github.com/l3msh0/keepeco/internal/db"
	"github.com/l3msh0/keepeco/internal/keychain"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var rootCmd = &cobra.Command{
	Use:   "keepeco",
	Short: "Select a password entry and copy its attribute.",
	Long:  `Select a password entry and copy its attribute.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbPath, err := homedir.Expand(args[0])
		if err != nil {
			abort("Failed to expand homedir", err)
		}

		finfo, err := os.Stat(dbPath)
		if err != nil {
			abort("Failed to stat file", err)
		}

		password, err := keychain.GetData(dbPath)
		hasEntry := (password != "")
		if err == nil {
			// nope
		} else if err == keychain.ErrorItemNotFound {
			fmt.Println("Password entry for the database must be created in the default keychain.")
			fmt.Printf("Enter password for %s: ", dbPath)
			bPassword, err := terminal.ReadPassword(int(syscall.Stdin))
			fmt.Println("")
			if err != nil {
				abort("Failed to read input", err)
			}
			password = string(bPassword)
		} else {
			abort("Failed to find database password from the default keychain. This may be resolved by creating a new keychain whose name is \"login\"", err)
		}

		errCh := make(chan error, 1)
		resultCh := make(chan struct {
			kdbx    *db.Database
			entries db.Entries
		}, 1)

		go func() {
			kdbx, err := db.Open(dbPath, password)
			if err != nil {
				errCh <- err
				close(errCh)
				return
			}
			if !hasEntry {
				err = keychain.Save(dbPath, password)
				if err != nil {
					fmt.Printf("[WARNING] Failed to save password: %s\n", err)
				}
			}
			resultCh <- struct {
				kdbx    *db.Database
				entries db.Entries
			}{kdbx, kdbx.Flatten()}
			close(resultCh)
		}()

		var kdbx *db.Database
		var entries db.Entries
		candidates, err := cache.Load(dbPath, password, finfo.ModTime())
		if err == cache.ErrCacheNotAvailable {
			select {
			case err := <-errCh:
				abort("Failed to open database", err)
			case r := <-resultCh:
				kdbx = r.kdbx
				entries = r.entries
				candidates = entries.Candidates()
				cache.Save(dbPath, password, finfo.ModTime(), candidates)
			}
		} else if err != nil {
			abort("Failed to load cache", err)
		} else {
			// WORKAROUND:
			//     Wait because fuzzyfinder.Find() hangs up when it is called
			//     immediately after a terminal launches and using cache.
			time.Sleep(75 * time.Millisecond)
		}

		i, err := fuzzyfinder.Find(candidates, func(i int) string {
			return candidates[i]
		})
		if err != nil {
			abort("Failed to select a candidate", err)
		}

		select {
		case err := <-errCh:
			abort("Failed to open database", err)
		case r, ok := <-resultCh:
			if ok {
				kdbx = r.kdbx
				entries = r.entries
			}
		}

		fmt.Printf("\"%s/%s\" selected.\n", entries[i].Prefix, entries[i].GetContent("Title"))
		fmt.Printf("Username: %s\n", entries[i].GetContent("UserName"))
		fmt.Printf("URL: %s\n", entries[i].GetContent("URL"))

		keyboard.Open()
		defer keyboard.Close()
		for {
			fmt.Println("")
			fmt.Println("Press ENTER to copy password and quit")
			fmt.Println("or continuous copy? [p]Password [u]UserName [U]URL")

			timer := time.After(180 * time.Second)
			cCh := make(chan rune)
			keyCh := make(chan keyboard.Key)
			go func(cCh chan<- rune, keyCh chan<- keyboard.Key) {
				c, key, err := keyboard.GetKey()
				if err != nil {
					abort("Failed to scan input", err)
				}
				if c != 0 {
					cCh <- c
				} else {
					keyCh <- key
				}
			}(cCh, keyCh)

			select {
			case <-timer:
				fmt.Println("Exit because there was no operation for 180 seconds")
				return
			case c := <-cCh:
				switch c {
				case 'u':
					clipboard.WriteAll(entries[i].GetContent("UserName"))
					fmt.Println("=> UserName copied.")
				case 'U':
					clipboard.WriteAll(entries[i].GetContent("URL"))
					fmt.Println("=> URL copied.")
				case 'p':
					kdbx.UnlockProtectedEntries()
					clipboard.WriteAll(entries[i].GetPassword())
					kdbx.LockProtectedEntries()
					fmt.Println("=> Password copied.")
				default:
					return
				}
			case key := <-keyCh:
				switch key {
				case keyboard.KeyEnter:
					kdbx.UnlockProtectedEntries()
					clipboard.WriteAll(entries[i].GetPassword())
					kdbx.LockProtectedEntries()
					fmt.Println("=> Password copied.")
					return
				default:
					return
				}
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.keepeco.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
// func initConfig() {
// 	// Find home directory.
// 	if cfgFile != "" {
// 		// Use config file from the flag.
// 		viper.SetConfigFile(cfgFile)
// 	} else {
// 		homeDir, err := homedir.Dir()
// 		if err != nil {
// 			abort("Failed to find home directory", err)
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}

// 		// Search config in home directory with name ".keepeco" (without extension).
// 		viper.AddConfigPath(homeDir)
// 		viper.SetConfigName(".keepeco")
// 	}

// 	viper.AutomaticEnv() // read in environment variables that match

// 	// If a config file is found, read it in.
// 	if err := viper.ReadInConfig(); err == nil {
// 		fmt.Println("Using config file:", viper.ConfigFileUsed())
// 	}
// }
