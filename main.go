package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"gopkg.in/yaml.v3"
)

const (
	appName    = "fxp"
	appVersion = "v0.1.0"
)

type ServerConfig struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Source      ServerConfig `yaml:"source"`
	Destination ServerConfig `yaml:"destination"`
	Files       []string     `yaml:"files"`
}

func main() {
	// command-line arguments
	configFilepath := flag.String("config", "./config.yaml", "config.yaml file location")
	printVersion := flag.Bool("version", false, "Prints this tool version")

	flag.Parse()

	if *printVersion {
		fmt.Printf("version: %s\n", appVersion)
		return
	}

	// verify the required fields
	if *configFilepath == "" {
		log.Println("error: Required flags are missing.")
		log.Println("Usage:")
		flag.PrintDefaults() // print default usage information
		os.Exit(1)
	}

	// load fron config file
	configData, err := os.ReadFile(*configFilepath)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	config := Config{}
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		log.Fatalf("error on unmarshal of config file. configFilename=%s, error=%s", *configFilepath, err.Error())
	}

	// verify files present
	if len(config.Files) == 0 {
		log.Fatalf("'files:' can not be empty on the yaml configuration, configFilename=%s", *configFilepath)
	}

	startTime := time.Now()

	// Connect to source FTP server
	sourceConn, err := ftp.Dial(config.Source.Address, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("failed to connect to source FTP, address=%s, error=%s", config.Source.Address, err.Error())
	}
	defer sourceConn.Quit()

	if err := sourceConn.Login(config.Source.Username, config.Source.Password); err != nil {
		log.Fatalf("source login failed, error=%s", err.Error())
	}

	// Connect to destination FTP server
	destConn, err := ftp.Dial(config.Destination.Address, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("failed to connect to destination FTP, address=%s, error=%s", config.Destination.Address, err.Error())
	}
	defer destConn.Quit()

	if err := destConn.Login(config.Destination.Username, config.Destination.Password); err != nil {
		log.Fatalf("destination login failed, error=%s", err.Error())
	}

	// transfer files
	for _, filename := range config.Files {
		if strings.HasSuffix(filename, "/") {
			// remove '/' from suffix
			dirName := strings.TrimSuffix(filename, "/")
			fxpDir(sourceConn, destConn, dirName)
		} else {
			err = fxpFile(sourceConn, destConn, filename)
			if err != nil {
				log.Fatalf("FXP transfer failed, filename=%s, error=%s", filename, err.Error())
			}
		}
	}

	log.Printf("overall timeTaken=%s\n", time.Since(startTime))
}

// list files from the given directory and call fxp file transfer operation
func fxpDir(sourceConn, destConn *ftp.ServerConn, dirName string) {
	entries, err := sourceConn.List(dirName)
	if err != nil {
		log.Printf("error on listing entries, directory=%s, error=%s\n", dirName, err.Error())
		return
	}
	for _, entry := range entries {
		switch entry.Type {
		case ftp.EntryTypeFile:
			filepath := fmt.Sprintf("%s/%s", dirName, entry.Name)
			err = fxpFile(sourceConn, destConn, filepath)
			if err != nil {
				log.Printf("FXP transfer failed, filename=%s, error=%s\n", filepath, err.Error())
				return
			}

		case ftp.EntryTypeFolder:
			dirpath := fmt.Sprintf("%s/%s", dirName, entry.Name)
			fxpDir(sourceConn, destConn, dirpath)
		}
	}
}

// perform fxp file transfer operation
func fxpFile(sourceConn, destConn *ftp.ServerConn, filename string) error {
	transferStartTime := time.Now()
	// retrieve file from source FTP
	resp, err := sourceConn.Retr(filename)
	if err != nil {
		log.Printf("failed to retrieve file from source, filename=%s, error=%s\n", filename, err.Error())
		return err
	}
	defer resp.Close()

	// store file on destination FTP
	if err := destConn.Stor(filename, resp); err != nil {
		log.Printf("failed to store file on destination, filename=%s, error=%s\n", filename, err.Error())
		return err
	}

	log.Printf("FXP transfer completed, filename=%s, timeTaken=%s", filename, time.Since(transferStartTime))
	return nil
}
