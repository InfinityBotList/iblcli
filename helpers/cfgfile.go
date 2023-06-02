package helpers

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func WriteConfig(name string, data any) error {
	cfgFile := ConfigFile()

	return Write(cfgFile+"/"+name, data)
}

func Write(fn string, data any) error {
	// Create config file
	f, err := os.Create(fn)

	if err != nil {
		return err
	}

	bytes, err := yaml.Marshal(data)

	if err != nil {
		return err
	}

	w, err := f.Write(bytes)

	if err != nil {
		return err
	}

	if os.Getenv("DEBUG") == "true" {
		fmt.Println("Write: wrote", w, "bytes to", fn)
	}

	return nil
}

func LoadConfig(name string, dst any) error {
	cfgFile := ConfigFile()

	if fsi, err := os.Stat(cfgFile + "/" + name); err != nil || fsi.IsDir() {
		return err
	} else {
		f, err := os.Open(cfgFile + "/" + name)

		if err != nil {
			return err
		}

		if os.Getenv("DEBUG") == "true" {
			fmt.Println("LoadConfig: opened", cfgFile+"/"+name)
		}

		defer f.Close()

		// Load into yaml
		err = yaml.NewDecoder(f).Decode(dst)

		if err != nil {
			if os.Getenv("DEBUG") == "true" {
				fmt.Println("LoadConfig: error decoding yaml:", err)
			}
			return err
		}
	}

	return nil
}
