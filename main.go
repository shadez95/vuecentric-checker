package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/urfave/cli"
)

var authors = []*cli.Author{
	{
		Name: "Dixon Begay",
	},
}

func openFileAndCheck(fileName string) {
	timeNow := time.Now().Local()

	// Setup log file
	logFileName := fmt.Sprintf("%v-%v-%v.log", timeNow.Year(), int(timeNow.Month()), timeNow.Day())
	curDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	f, err := os.OpenFile(filepath.Join(curDir, "logs", logFileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	// Set logging output to the above log file
	log.SetOutput(f)

	// Create the waitgroup that will count number of goroutines
	var wg sync.WaitGroup

	// Open the file that was passed as an argument to the program
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Setup the reader for the file that was passed as an argument to the program
	scanner := bufio.NewScanner(file)

	// log.Println("------------ Starting to check if vuecentric service is running on computers ------------")
	// Not logging the above anymore. Isn't useful and was originally there for debugging. 4/9/2020

	// Start reading file line by line
	for scanner.Scan() {
		go func(computerName string) {
			// Add 1 to the waitgroup for this goroutine
			wg.Add(1)

			//Connect to remote computer
			// log.Printf("[%s] Connecting...\n", computerName)
			// Not logging the above anymore. Isn't useful and was originally there for debugging. 4/9/2020
			computer, err := mgr.ConnectRemote(computerName)
			if err != nil {
				log.Printf("[%s] %v", computerName, err)
			} else {

				// Get the lock status of the service controller
				lockStatus, err := computer.LockStatus()
				if err != nil {
					log.Printf("[%s] %v", computerName, err)
				}

				// Check if the service controller is locked by, and if so, by who, and for how long
				if lockStatus.IsLocked {
					log.Println(fmt.Sprintf("[%s] Locked by %s for %v", computerName, lockStatus.Owner, lockStatus.Age))
				} else {

					// Get the vuecentric service
					vcUpdaterSvc, err := computer.OpenService("vcUpdater")
					if err != nil {
						log.Printf("[%s] %v", computerName, err)
					} else {

						// Delcare svcStatus here so we are not making declarations every loop
						var svcStatus svc.Status
						svcStatus.State = svc.State(0)

						// Error counter for keeping track of how many attempts are made to query the vcUpdater service
						errCount := 0

						// Keep looping until the service is running
					queryLoop:
						for {
							// Get the vcUpdater service
							svcStatus, err = vcUpdaterSvc.Query()
							if err != nil {
								log.Printf("[%s] %v", computerName, err)
								errCount++
								if errCount > 2 {
									log.Printf("[%s] Failed to query vcUpdater service after %v attempts\n", computerName, errCount)
									break queryLoop
								}
							} else {

								switch svcStatus.State {
								case svc.Stopped:
									log.Printf("[%s] vcUpdater is stopped\n", computerName)

									// If the service is stopped then start the vcUpdater service
									if err = vcUpdaterSvc.Start(); err != nil {
										log.Printf("[%s] %v", computerName, err)
										break queryLoop
									}

									log.Printf("[%s] starting vcUpdater\n", computerName)

								case svc.StartPending:
									log.Printf("[%s] vcUpdater is starting up\n", computerName)

								case svc.StopPending:
									log.Printf("[%s] vcUpdater is stopping\n", computerName)

								case svc.Running:
									log.Printf("[%s] vcUpdater is running\n", computerName)
									break queryLoop

								case svc.ContinuePending:
									log.Printf("[%s] vcUpdater is starting up\n", computerName)

								case svc.PausePending:
									log.Printf("[%s] vcUpdater is pausing\n", computerName)

								case svc.Paused:
									log.Printf("[%s] vcUpdater is paused\n", computerName)

									_, err = vcUpdaterSvc.Control(svc.Continue)
									if err != nil {
										log.Printf("[%s] %v", computerName, err)
									}

									log.Printf("[%s] resuming vcUpdater\n", computerName)

								}
							}

							// Wait 2 seconds so computer and network isn't spammed with requests
							time.Sleep(time.Second * 1)
						}
					}
				}
				// log.Printf("[%s] Disconnecting...\n", computerName)
				// Not logging the above anymore. Isn't useful and was originally there for debugging. 4/9/2020
				// Disconnect from the computer
				computer.Disconnect()
			}

			// Reduce the counter for the waitgroup
			wg.Done()
		}(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// Allow program to run through all goroutines before exiting
	wg.Wait()
	// log.Println("------------ Finished checking computers ------------")
	// Not logging the above anymore. Isn't useful and was originally there for debugging. 4/9/2020
}

func main() {

	app := &cli.App{
		Name:      "vuecentric-checker",
		Usage:     "vuecentric-checker ensures VueCentric service is running for a list of Windows computers.",
		UsageText: "vuecentric-checker <ListOfComputers> [Required] (ListOfComputers is a file containing a list of remote computers on a network)",
		ArgsUsage: "ListOfComputers (file containing a list of remote computers on a network)",
		Copyright: "Copyright (c) 2020 Dixon Begay aka shadez95",
		Authors:   authors,
		Action: func(c *cli.Context) error {
			// Check if args present first. If not, throw up helper info
			if c.Args().Present() {
				openFileAndCheck(c.Args().First())
				return nil
			}

			// If no argument is passed, then show the app help info
			cli.ShowAppHelp(c)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}
