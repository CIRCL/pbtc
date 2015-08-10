// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/gocql/gocql"
)

var session *gocql.Session

func main() {
	// set the parameters for our cassandra connection
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.DiscoverHosts = true
	cluster.DefaultTimestamp = false

	// establish the cassandra session
	session, err := cluster.CreateSession()
	if err != nil {
		fmt.Println("could not create session")
		os.Exit(1)
	}
	defer session.Close()

	// get a list of iles in the logs folder
	files, err := ioutil.ReadDir("../logs")
	if err != nil {
		fmt.Println("could not read logs folder")
		os.Exit(1)
	}

	// iterate through files for processing
	for _, file := range files {
		err := process(file)
		if err != nil {
			fmt.Printf("%v", err)
		}
	}

	os.Exit(0)
}

func process(file os.FileInfo) error {
	// ignore all directories
	if file.IsDir() {
		return fmt.Errorf("can't process directory: %v", file.Name())
	}

	// ignore all non-log files
	if path.Ext(file.Name()) != ".txt" {
		return fmt.Errorf("can only process log files: %v", file.Name())
	}

	// open file
	f, err := os.Open(file.Name())
	if err != nil {
		return fmt.Errorf("could not open file: %v", file.Name())
	}
	defer f.Close()

	// try to use scanner to read first line
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return fmt.Errorf("could not read first line: %v", file.Name())
	}

	// check first line header for log version
	line := scanner.Text()
	if line != "PBTC Log Version 1" {
		return fmt.Errorf("unknown header for file: %v (%v)", f.Name(), line)
	}

	// reset file pointer
	_, err = f.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("could not reset file pointer: %v", f.Name())
	}

	// stream file into hasher to get fingerprint
	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return fmt.Errorf("could not stream file data into hasher: %v", f.Name())
	}

	// get fingerprint hash and check for duplicate
	hash := hasher.Sum(nil)
	fmt.Printf("hash for file: %v - %v", f.Name(), hash)

	// import file into cassandra
	scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		err := insert(scanner.Text())
		if err != nil {
			fmt.Printf("%v", err)
		}
	}

	return nil
}

func insert(line string) error {
	return nil
}
