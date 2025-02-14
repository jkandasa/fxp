package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
	"gopkg.in/yaml.v3"
)

const (
	appName    = "fxp"
	appVersion = "v0.2.1"

	defaultConnectionTimeout  = time.Second * 5
	defaultFxpTransferTimeout = time.Minute * 10
)

type ServerConfig struct {
	IsTLS    bool   `yaml:"is_tls"`
	Insecure bool   `yaml:"insecure"`
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Debug    bool   `yaml:"debug"`
}

type Config struct {
	IsMDTM             bool         `yaml:"is_mdtm"`
	ConnectionTimeout  string       `yaml:"connection_timeout"`
	FXPTransferTimeout string       `yaml:"fxp_transfer_timeout"`
	Source             ServerConfig `yaml:"source"`
	Destination        ServerConfig `yaml:"destination"`
	Files              []string     `yaml:"files"`
}

var (
	logger             *customLogger
	connectionTimeout  = defaultConnectionTimeout
	fxpTransferTimeout = defaultFxpTransferTimeout
)

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

	logger = getCustomLogger("APP")
	startTime := time.Now()

	_timeout, err := time.ParseDuration(config.ConnectionTimeout)
	if err == nil {
		connectionTimeout = _timeout
	}

	_timeout, err = time.ParseDuration(config.FXPTransferTimeout)
	if err == nil {
		fxpTransferTimeout = _timeout
	}

	// Connect to source FTP server
	sourceConn, err := getFtpConnection(config, config.Source, "source FTP")
	if err != nil {
		log.Fatalf("failed to connect to source FTP, address=%s, error=%s", config.Source.Address, err.Error())
	}
	defer sourceConn.Quit()

	if err := sourceConn.Login(config.Source.Username, config.Source.Password); err != nil {
		log.Fatalf("source login failed, error=%s", err.Error())
	}

	// Connect to destination FTP server
	destConn, err := getFtpConnection(config, config.Destination, "dest FTP")
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
			err = startFXP(sourceConn, destConn, filename)
			if err != nil {
				log.Fatalf("FXP transfer failed, filename=%s, error=%s", filename, err.Error())
			}
		}
	}

	logger.Printf("overall timeTaken=%s\n", time.Since(startTime))
}

func getFtpConnection(config Config, server ServerConfig, ftpName string) (*ftp.ServerConn, error) {
	options := []ftp.DialOption{
		ftp.DialWithTimeout(connectionTimeout),
		ftp.DialWithWritingMDTM(config.IsMDTM),
	}

	if server.IsTLS {
		tls := &tls.Config{}
		tls.InsecureSkipVerify = server.Insecure
		options = append(options, ftp.DialWithTLS(tls))
	}

	if server.Debug {
		options = append(options, ftp.DialWithDebugOutput(getCustomLogger(ftpName)))
	}

	return ftp.Dial(server.Address, options...)
}

// list files from the given directory and call fxp file transfer operation
func fxpDir(sourceConn, destConn *ftp.ServerConn, dirName string) {
	entries, err := sourceConn.List(dirName)
	if err != nil {
		logger.Printf("error on listing entries, directory=%s, error=%s\n", dirName, err.Error())
		return
	}
	for _, entry := range entries {
		switch entry.Type {
		case ftp.EntryTypeFile:
			filepath := fmt.Sprintf("%s/%s", dirName, entry.Name)
			err = startFXP(sourceConn, destConn, filepath)
			if err != nil {
				logger.Printf("FXP transfer failed, filename=%s, error=%s\n", filepath, err.Error())
				return
			}

		case ftp.EntryTypeFolder:
			dirpath := fmt.Sprintf("%s/%s", dirName, entry.Name)
			fxpDir(sourceConn, destConn, dirpath)
		}
	}
}

func createDirs(destConn *ftp.ServerConn, filename string) error {
	// create directory if not available
	targetDir := path.Dir(filename)

	dirs := strings.Split(targetDir, "/")
	currentPath := ""

	for _, dir := range dirs {
		currentPath = path.Join(currentPath, dir)
		// ignore the error
		destConn.MakeDir(currentPath)
	}
	return nil
}

func startFXP(sourceConn, destConn *ftp.ServerConn, filename string) error {
	// create dirs
	if err := createDirs(destConn, filename); err != nil {
		return err
	}

	fileStartTime := time.Now()
	logger.Printf("initiating FXP transfer, filename:%s\n", filename)
	err := transferFileFXP(sourceConn, destConn, filename)
	if err != nil {
		logger.Printf("failed to retrieve file from source, filename=%s, error=%s\n", filename, err.Error())
		return err
	}
	logger.Printf("FXP transfer completed, filename=%s, timeTaken=%s\n", filename, time.Since(fileStartTime))

	return nil
}

// source: https://en.wikipedia.org/wiki/File_eXchange_Protocol
// move the dest FTP server in to passive mode, returns ip and port details to connect
// create a connection from source server to dest server with the given ip and port
// post the command on 'dest' server: 'STOR filename' on dest server
// post the command on 'source' server: 'RETR filename' on source server
// monitor logs from both servers
func transferFileFXP(source, dest *ftp.ServerConn, filename string) error {
	// enable passive mode on the destination
	_, line, err := dest.Cmd(ftp.StatusPassiveMode, "PASV")
	if err != nil {
		return err
	}

	// PASV response format : 227 Entering Passive Mode (h1,h2,h3,h4,p1,p2).
	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start == -1 || end == -1 {
		return errors.New("invalid PASV response format")
	}
	sourceAddr := line[start+1 : end]

	_, _, err = source.Cmd(ftp.StatusCommandOK, "PORT %s", sourceAddr)
	if err != nil {
		return err
	}

	_, err = dest.GetConn().Cmd("STOR %s", filename)
	if err != nil {
		return err
	}

	_, err = source.GetConn().Cmd("RETR %s", filename)
	if err != nil {
		return err
	}

	// wait until the FXP transfer completes
	timeout := time.Now().Add(fxpTransferTimeout)
	isSourceDone := false
	isDestDone := false
	for {

		if !isDestDone {
			msg, err := dest.GetConn().ReadLine()
			if err != nil {
				logger.Printf("error on receiving message from dest, error:%s\n", err.Error())
			}
			if strings.Contains(msg, "226") {
				isDestDone = true
			} else if !strings.Contains(msg, "150") {
				return errors.New(msg)
			}
		}

		if !isSourceDone {
			msg, err := source.GetConn().ReadLine()
			if err != nil {
				logger.Printf("error on receiving message from source, error:%s\n", err.Error())
			}
			if strings.Contains(msg, "226") {
				isSourceDone = true
			} else if !strings.Contains(msg, "150") {
				return errors.New(msg)
			}
		}

		if isDestDone && isSourceDone {
			return nil
		}

		if time.Now().After(timeout) {
			logger.Printf("reached fxp transfer timeout\n")
			return errors.New("reached fxp transfer timeout")
		}
	}
}

// custom logger
type customLogger struct {
	prefix string
	buffer bytes.Buffer
	mutex  sync.Mutex
}

func (cl *customLogger) Write(data []byte) (n int, err error) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	for index, b := range data {
		if b == '\n' {
			log.Printf("[%10s] %s", cl.prefix, cl.buffer.String())
			cl.buffer.Reset()
			continue
		}
		if err := cl.buffer.WriteByte(b); err != nil {
			return index, err
		}
	}
	return len(data), nil
}

func (cl *customLogger) Printf(format string, v ...any) {
	log.Printf("[%10s] %s", cl.prefix, fmt.Sprintf(format, v...))
}

func getCustomLogger(prefix string) *customLogger {
	return &customLogger{
		prefix: prefix,
		buffer: bytes.Buffer{},
	}
}
