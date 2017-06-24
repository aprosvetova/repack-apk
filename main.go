package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/rsc/zipmerge/zip"
)

// Config ...
type Config struct {
	CertName           string // TEST.SF and TEST.RSA
	PrivateKeyPEM      string // /path/to/private_key.pem
	SourceAPK          string //  my-bucket/origin.apk
	DestAPK            string // my-bucket/dest.apk
	CPIDFile           string // /path/to/cpid
	OSSEndpoint        string
	OSSAccessKeyID     string
	OSSAccessKeySecret string
	WorkDir            string // working dir to save temp files
}

func (c Config) String() string {
	buf, _ := json.MarshalIndent(c, "", "  ")
	return string(buf)
}

var g Config

func init() {
	flag.StringVar(&g.CertName, "cert-name", "", "cert name for .SF and .RSA")
	flag.StringVar(&g.PrivateKeyPEM, "priv-pem", "", "private key pem")
	flag.StringVar(&g.SourceAPK, "source", "", "source apk")
	flag.StringVar(&g.DestAPK, "dest", "", "dest apk")
	flag.StringVar(&g.CPIDFile, "cpid", "", "cpid file")
	flag.StringVar(&g.OSSEndpoint, "oss-ep", "", "oss endpoint")
	flag.StringVar(&g.OSSAccessKeyID, "oss-id", "", "oss access key id")
	flag.StringVar(&g.OSSAccessKeySecret, "oss-key", "", "oss access key secret")
	flag.StringVar(&g.WorkDir, "work-dir", "", "working dir")
}

// print error and exit
func perror(msg string, args ...interface{}) {
	log.Printf(msg, args...)
	os.Exit(1)
}

func main() {
	flag.Parse()
	log.Printf("using config: %s", g.String())

	ossReader, err := NewReader(
		g.OSSEndpoint, g.OSSAccessKeyID, g.OSSAccessKeySecret, g.SourceAPK)
	if err != nil {
		perror("oss reader: %v", err)
	}
	objectSize, err := ossReader.Size()
	if err != nil {
		perror("object size: %v", err)
	}

	zipReader, err := zip.NewReader(ossReader, objectSize)
	if err != nil {
		perror("zip reader: %v", err)
	}

	err = changeManifest(zipReader)
	if err != nil {
		perror("change manifest: %v", err)
	}

	ossWriter, err := NewWriter(
		g.OSSEndpoint, g.OSSAccessKeyID, g.OSSAccessKeySecret,
		g.DestAPK, g.SourceAPK, zipReader.AppendOffset())
	if err != nil {
		perror("oss writer: %v", err)
	}
	defer func() {
		err := ossWriter.Flush()
		if err != nil {
			perror("flush oss: %v", err)
		}
	}()

	writer := zipReader.Append(ossWriter)
	defer writer.Close()

	// copy cpid file
	if err := copyFile(writer, "cpid", g.CPIDFile); err != nil {
		perror("copy cpid: %v", err)
	}
	// copy meta files: MANIFEST.MF/CERT.SF/CERT.RSA
	if err := copyMeta(writer); err != nil {
		perror("copy meta: %v", err)
	}
}