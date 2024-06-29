package config

import (
	"os"
	"path/filepath"
)

var (
	CAFile         string
	ServerCertFile string
	ServerKeyFile  string
	ClientCertFile string
	ClientKeyFile  string
)

func init() {
	projectRoot, err := findProjectRoot(".cert")
	if err != nil {
		panic("Could not find .cert folder in project root: " + err.Error())
	}

	CAFile = certPath(projectRoot, "ca.pem")
	ServerCertFile = certPath(projectRoot, "server.pem")
	ServerKeyFile = certPath(projectRoot, "server-key.pem")
	ClientCertFile = certPath(projectRoot, "client.pem")
	ClientKeyFile = certPath(projectRoot, "client-key.pem")
}

func certPath(projectRoot, fileName string) string {
	return filepath.Join(projectRoot, ".cert", fileName)
}

func findProjectRoot(target string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, target)); !os.IsNotExist(err) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}
