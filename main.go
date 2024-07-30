package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// Used example code from https://pkg.go.dev/go.mau.fi/whatsmeow
	dbLog := waLog.Stdout("Database", "ERROR", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:login.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	params := whatsmeow.GetProfilePictureParams{
		Preview:     false,
		ExistingID:  "",
		IsCommunity: false,
	}

	user := client.Store.ID.User
	server := client.Store.ID.Server

	println("Using:", user)
	jid, err := types.ParseJID(user + "@" + server)
	checkErr(err)

	//get url for current pfp
	info, err := client.GetProfilePictureInfo(jid, &params)
	checkErr(err)

	println("\nOld PFP:", info.URL)

	image, err := os.ReadFile("pfp.jpg")
	if err != nil {
		println("Picture should be a jpeg, and named pfp.jpg")
		panic(err)
	}

	var answer string
	fmt.Print("\nDISCLAIMER: This program is intended for educational use only. Users must comply with WhatsApp's terms of service and community guidelines. The authors of this program are not affiliated with WhatsApp Inc., and this program is not endorsed or approved by WhatsApp Inc.\nWould you like to continue (y/n): ")
	_, err = fmt.Scanln(&answer)
	checkErr(err)

	if answer != "y" {
		client.Disconnect()
		os.Exit(0)
	}

	//update pfp
	_, err = client.SetGroupPhoto(types.EmptyJID, image)
	checkErr(err)

	println("\nUpdated profile picture")

	//get url for new pfp
	info, err = client.GetProfilePictureInfo(jid, &params)
	checkErr(err)

	println("\nNew PFP:", info.URL)

	//wait
	println("\nDone!")
	_, _ = fmt.Scanln(&answer)

	client.Disconnect()
	os.Exit(0)

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
