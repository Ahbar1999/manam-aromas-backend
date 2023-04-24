package main

import (
	"io"
	"fmt"
	"net/http"
	"log"
	"encoding/json"
	"net/url"
	"context"
	"os"	
	"github.com/jackc/pgx/v5/pgxpool"	
	"github.com/jackc/pgx/v5"
	"strings"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var response []byte 
	if r.Method == "GET" {
		parsed, _ := url.Parse(r.RequestURI)
		q := parsed.Query()
		response, _ = json.Marshal(map[string]string{"hello": strings.Join(q["name"], "")})	
	} else {
		response = []byte("Recieved post request")	
		// fmt.Println("Recieved Request of type: ", r.Method)	
	}
	// use encoder object with a writer to encode and send data 
	encoder := json.NewEncoder(w)
	encoder.Encode(string(response))	
	// not using io writer which allows buffering
	// io.WriteString(w, string(response))	
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
		case "POST":
			file, fileHeader, err := r.FormFile("file")	
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldnt parse file: %v\n ... Exiting", err)
				return 	
			}
			fmt.Println("Recieved file....")
			fmt.Println("Filename: ", fileHeader.Filename)
			fmt.Println("File size: ", fileHeader.Size)
			fmt.Println("Reading file: ")
			filepath, _ := os.Getwd()
			filepath += fmt.Sprintf("/tmp/%s", fileHeader.Filename) 
			f, err := os.Create(filepath)
			defer f.Close()	
			
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v, Cannot create a new file.. Skipping Store\n", err)
			}

			// read files
			buf := make([]byte, 1024)
			for ;; {
				n, err := file.Read(buf)
				if err == io.EOF {
					fmt.Println("\nEnd of File")	
					break	
				}	
				if err != nil {
					fmt.Fprintf(os.Stderr, "Cannot read file.. Exiting\n")	
					return	
				}
				if n > 0 {
					fmt.Print(string(buf[:n]))
					_, err := f.Write(buf[:n])
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v, Couldnt write to file on server... Continuing without\n", err) 	
					}
				}
			}		
			f.Sync()

			// 200 OK
			w.WriteHeader(200)
			w.Write([]byte("File read successfully!"))
			
		case "GET":
			w.WriteHeader(404)
			w.Write([]byte("Only POST method is allowed"))
	}
}



func runServer() {
	fmt.Println("Running server, listening on port: 8080")
	// http.ListenAndServe(":8080", nil)

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)	
	log.Fatal(http.ListenAndServe(":8080", nil))	
}

func connDb() {
	URI := "postgresql://localhost:5432/postgres?user=postgres&password=1234";
	dbpool, err := pgxpool.New(context.Background(), URI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(dbpool.Ping(context.Background()))	
	
	// defer terminating pool until the end of main
	defer dbpool.Close()

	// var greeting string	
	var rows pgx.Rows
	rows, err = dbpool.Query(context.Background(), "SELECT * FROM \"Posts\"")
	if err != nil {
		fmt.Println("Error occured while querying: \n", err);	
	}	
	
	// read return of query
	for ;; {
		if prs := rows.Next(); prs {
			values, _ := rows.Values()
			fmt.Println(values)
		} else {
			// if no more values present
			break	
		}	
	} 
}

func main() {
	// start server 
	go runServer()
	// go connDb()
	
	// wait for user input (to avoid instant exit)
	var res string
	fmt.Scan(&res)
}
