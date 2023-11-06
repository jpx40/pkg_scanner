package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	//_ "github.com/marcboeker/go-duckdb"
	// _ "github.com/marcboeker/go-duckdb"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type pkg struct {
	name         string
	version      string
	dependencies []string
}

func main() {
	t0 := time.Now()
	read_packages()
	t1 := time.Now()
	fmt.Printf("The call took %v to run.\n", t1.Sub(t0))
}

func read_packages() []pkg {
	cmdStruct := exec.Command("pacman", "-Qqe")
	// out, err := cmdStruct.Output()
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmdStruct.Stdout = &out
	cmdStruct.Stderr = &out

	err := cmdStruct.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}
	r := strings.ReplaceAll(out.String(), "\n", ",")

	s := strings.Split(r, ",")

	var pkgs []pkg

	// i := 0
	db := db_connect()

	for _, pkg_tmp := range s {

		version := get_version(pkg_tmp)
		depends := get_depends(pkg_tmp)

		// insert into database
		if err != nil {
			panic(err)
		}

		Insert(db, pkg{name: pkg_tmp, version: version, dependencies: depends})
		// pkgs = append(pkgs, pkg{name: pkg_tmp, version: version, depends: depends})
		// i++
		// if i > 10 {
		// 	break
		// }
	}

	return pkgs
}

func get_version(pkg_tmp string) string {
	cmdStruct := exec.Command("pacman", "-Q", pkg_tmp)

	// out, err := cmdStruct.Output()
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmdStruct.Stdout = &out
	cmdStruct.Stderr = &out

	err := cmdStruct.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}
	v := strings.Split(out.String(), " ")
	// version := v[1]
	return v[1]
}

func get_depends(pkg_tmp string) []string {
	cmdStruct := exec.Command("expac", "-S", "'%D'", pkg_tmp)
	// out, err := cmdStruct.Output()
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmdStruct.Stdout = &out
	cmdStruct.Stderr = &out

	err := cmdStruct.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}
	r := strings.ReplaceAll(out.String(), "'", "")

	r = strings.ReplaceAll(r, "\n", ",")
	r = strings.ReplaceAll(r, " ", ",")
	depends := strings.Split(r, ",")

	depends = deleteEmptyStrings(depends)
	depends = double_appears(depends)

	return depends
}

func db_connect() *sql.DB {
	// c := "postgres://postgres:artemis34@127.17.0.2:5432//postgres"
	c := "host=0.0.0.0 user=jonas password=artemis34 dbname=postgres port=32768 sslmode=disable"

	conn, err := sql.Open("pgx", c)
	if err != nil {
		panic("error opening postgres connection: " + err.Error())
	}

	return conn
}

// insert
func Insert(c *sql.DB, d pkg) {
	name := d.name
	version := d.version
	dependencies := to_json(d.dependencies)
	_, err := c.Exec("INSERT INTO arch_packages (name, version, dependencies) VALUES ($1, $2, $3)", name, version, dependencies)
	if err != nil {
		panic("error inserting data: " + err.Error())
	}
}

// to json

func to_json(s []string) string {
	bytes, _ := json.Marshal(s)

	// write to file
	f, err := os.OpenFile("test.json", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	_, err = f.WriteString(string(bytes) + "\n")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	return string(bytes)
}

func remove(s []int, i int) []int {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func to_bytes(s []string) []byte {
	wbuf := new(bytes.Buffer)
	err := binary.Write(wbuf, binary.LittleEndian, s)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return wbuf.Bytes()
}

func deleteEmptyStrings(strings []string) []string {
	// Create a new slice to store the non-empty strings.
	nonEmptyStrings := []string{}

	// Iterate over the original slice and add each non-empty string to the new slice.
	for _, s := range strings {
		if s != "" {
			if s != " " {
				nonEmptyStrings = append(nonEmptyStrings, s)
			}
		}
	}

	// Return the new slice.
	return nonEmptyStrings
}

// delete double string in slice
func double_appears(s []string) []string {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] == s[j] {
				s = append(s[:j], s[j+1:]...)
				j--
			}
		}
	}
	return s
}
