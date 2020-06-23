package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/urfave/cli"
)

var authors = []*cli.Author{
	{
		Name: "Dixon Begay",
	},
}

// Path to database (sqlite)
var dbPath string

// File path containing list of computers
var filePath string

// Database file
var db *gorm.DB

func openFileAndCheck() {

	db, err := gorm.Open("sqlite3", dbPath)
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Computer{}, &VcError{})

	defer db.Close()

	// Create the waitgroup that will count number of goroutines
	var wg sync.WaitGroup

	// Open the file that was passed as an argument to the program
	file, err := os.Open(filePath)
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

									computer := Computer{Name: computerName}
									// db.FirstOrCreate(&computer, Computer{Name: computerName})
									db.Where(Computer{Name: computerName}).FirstOrCreate(&computer)
									// db.Model(&computer).Update("vcErrors", append(computer.VcErrors, VcError{DateTime: time.Now()}))
									vcError := VcError{
										Computer: computerName,
										Status:   "stopped",
									}
									db.Save(computer)
									db.Model(&computer).Association("VcErrors").Append(vcError)
									db.Save(computer)
									db.Save(vcError)

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

									computer := Computer{Name: computerName}
									// db.FirstOrCreate(&computer, Computer{Name: computerName})
									db.Where(Computer{Name: computerName}).FirstOrCreate(&computer)
									// db.Model(&computer).Update("vcErrors", append(computer.VcErrors, VcError{DateTime: time.Now()}))
									vcError := VcError{
										Computer: computerName,
										Status:   "paused",
									}
									db.Save(computer)
									db.Model(&computer).Association("VcErrors").Append(vcError)
									db.Save(computer)
									db.Save(vcError)

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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dbPath",
				Aliases:     []string{"db"},
				Usage:       "Path to database file (sqlite3). If no path is specified, memory will be used.",
				Value:       ":memory:",
				TakesFile:   true,
				Destination: &dbPath,
			},
			&cli.StringFlag{
				Name:        "computerList",
				Usage:       "Path to file containing list of remote computers on a network.",
				TakesFile:   true,
				Destination: &filePath,
				Required:    true,
			},
		},
		Name:      "vuecentric-checker",
		Usage:     "vuecentric-checker ensures VueCentric service is running for a list of Windows computers.",
		UsageText: "vuecentric-checker ensures VueCentric service is running for a list of Windows computers.",
		ArgsUsage: "ListOfComputers (file containing a list of remote computers on a network)",
		Copyright: "Copyright (c) 2020 Dixon Begay aka shadez95",
		Authors:   authors,
		Action: func(c *cli.Context) error {

			// Check if flags are present first. If not, continue and throw up helper info
			if c.NumFlags() > 0 {
				openFileAndCheck()
			}
			return nil
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}
