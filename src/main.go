package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	gosxnotifier "github.com/deckarep/gosx-notifier"
	"github.com/olekukonko/tablewriter"
)

const (
	apiURL  = "http://api.aladhan.com/v1/timingsByCity"
	city    = "Boynton Beach"
	country = "United States"
	method  = 3 // Muslim World League method
)

type Timings struct {
	Fajr     string `json:"Fajr"`
	Sunrise  string `json:"Sunrise"`
	Dhuhr    string `json:"Dhuhr"`
	Asr      string `json:"Asr"`
	Sunset   string `json:"Sunset"`
	Maghrib  string `json:"Maghrib"`
	Isha     string `json:"Isha"`
	Imsak    string `json:"Imsak"`
	Midnight string `json:"Midnight"`
}

type Data struct {
	Timings Timings `json:"timings"`
}

type Response struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

func getPrayerTimes() (Timings, error) {
	url := fmt.Sprintf("%s?city=%s&country=%s&method=%d", apiURL, city, country, method)
	resp, err := http.Get(url)
	if err != nil {
		return Timings{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Timings{}, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return Timings{}, err
	}

	return response.Data.Timings, nil
}

func getNextPrayerTime(timings Timings) (string, string) {
	currentTime := time.Now().Format("15:04")

	prayers := []struct {
		Name string
		Time string
	}{
		{"Fajr", timings.Fajr},
		{"Sunrise", timings.Sunrise},
		{"Dhuhr", timings.Dhuhr},
		{"Asr", timings.Asr},
		{"Maghrib", timings.Maghrib},
		{"Isha", timings.Isha},
	}

	for _, prayer := range prayers {
		if currentTime < prayer.Time {
			return prayer.Name, prayer.Time
		}
	}

	// If all prayers have passed, return the first prayer of the next day
	return prayers[0].Name, timings.Fajr
}

func showNotification(title, message string) {
	note := gosxnotifier.NewNotification(title)

	//Optionally, set a title
	note.Title = title

	//Optionally, set a subtitle
	note.Subtitle = message

	//Optionally, set a sound from a predefined set.
	note.Sound = gosxnotifier.Basso

	//Optionally, set a group which ensures only one notification is ever shown replacing previous notification of same group id.
	note.Group = "github.iustusae.adhan"

	//Optionally, set a sender (Notification will now use the Safari icon)
	//note.Sender = "com.apple.Safari"

	//Optionally, specifiy a url or bundleid to open should the notification be
	//clicked.
	//note.Link = "http://www.yahoo.com" //or BundleID like: com.apple.Terminal

	//Optionally, an app icon (10.9+ ONLY)
	note.AppIcon = "mosque.png"

	//Optionally, a content image (10.9+ ONLY)
	note.ContentImage = "mosque.jpeg"

	//Then, push the notification
	err := note.Push()

	//If necessary, check error
	if err != nil {
		log.Println("Uh oh!")
	}
}

func printTable(header []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetAlignment(tablewriter.ALIGN_CENTER)

	for _, row := range data {
		table.Append(row)
	}

	table.Render()
}

func handleUserInput() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a command (or 'q' to quit): ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)

		switch command {
		case "next":
			timings, err := getPrayerTimes()
			if err != nil {
				log.Println("Failed to fetch prayer times:", err)
				continue
			}

			nextPrayer, nextTime := getNextPrayerTime(timings)
			fmt.Printf("Next prayer: %s, Time: %s\n", nextPrayer, nextTime)
		case "all":
			timings, err := getPrayerTimes()
			if err != nil {
				log.Println("Failed to fetch prayer times:", err)
				continue
			}

			header := []string{"Prayer", "Time"}
			data := [][]string{
				{"Fajr", timings.Fajr},
				{"Sunrise", timings.Sunrise},
				{"Dhuhr", timings.Dhuhr},
				{"Asr", timings.Asr},
				{"Maghrib", timings.Maghrib},
				{"Isha", timings.Isha},
			}
			printTable(header, data)
		case "q":
			os.Exit(0)
			return
		default:
			fmt.Println("Invalid command")
		}
	}
}

func checkPrayerTimes(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		timings, err := getPrayerTimes()
		if err != nil {
			log.Println("Failed to fetch prayer times:", err)
			time.Sleep(time.Minute) // Retry after a minute
			continue
		}

		nextPrayer, nextTime := getNextPrayerTime(timings)
		fmt.Printf("Next prayer: %s, Time: %s\n", nextPrayer, nextTime)

		// Check if the current time matches the next prayer time
		currentTime := time.Now().Format("15:04")
		if currentTime == nextTime {
			showNotification("Prayer Time", fmt.Sprintf("It's time for %s prayer.", nextPrayer))
		}

		time.Sleep(1 * time.Minute) // Check every minute
	}
}

func main() {
	showNotification("Adhan", "Adhan app is active!")
	time.Sleep(3 * time.Second)
	timings, err := getPrayerTimes()
	if err != nil {
		log.Println("Failed to fetch prayer times:", err)
		time.Sleep(time.Minute)
	}
	nx, tim := getNextPrayerTime(timings)
	showNotification("Adhan", "Next Prayer is : "+nx+" at: "+tim)
	var wg sync.WaitGroup
	wg.Add(1)

	go checkPrayerTimes(&wg)
	handleUserInput()

	wg.Wait()
}
