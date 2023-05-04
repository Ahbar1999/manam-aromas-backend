package main

import (
	"io"
	"fmt"
	"bytes"
	"net/http"
	"log"
	"encoding/json"
	"net/url"
	"context"
	"os"	
	"github.com/jackc/pgx/v5/pgxpool"	
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
	"path/filepath"
	// "github.com/golang-jwt/jwt/v4"	
)

/*
	TODO:
*/


var URI string = "postgresql://localhost:5432/postgres?user=postgres&password=1234"
var dbpool *pgxpool.Pool

func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		// 404- Bad Request
		w.WriteHeader(400)
		w.Write([]byte("make GET req with \"filepath\" parameter"))
		return
	}

	parsed, _ := url.Parse(r.RequestURI)
	q := parsed.Query()
	fn := strings.Join(q["filepath"], "")
	fmt.Println("Sending file with filename: ")
	fmt.Println(filepath.Base(fn))
	data, err := os.ReadFile(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v while reading filepath: %v\n", err, fn)  
	}
	
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(fn)))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, bytes.NewReader(data))	
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var response []byte

	if r.Method == "GET" {
		parsed, _ := url.Parse(r.RequestURI)
		q := parsed.Query()
		// use queryDb() to get
		
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
				fmt.Fprintf(os.Stderr, "Couldnt parse file: %v... Exiting\n", err)
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
			// commit the data on the disk
			f.Sync()

			// 200 OK
			w.WriteHeader(200)
			w.Write([]byte("File read successfully!"))
			
		case "GET":
			// 404: ERROR: RESOURCE NOT FOUND
			w.WriteHeader(404)
			w.Write([]byte("Only POST method is allowed"))
	}
}

func listReports(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT * FROM "Reports"
	`
	// currently this just returns nil	
	result := queryDb(query)	
	encoder := json.NewEncoder(w)
	// ok
	w.WriteHeader(200)
	// fmt.Println("Recieved result: ")
	fmt.Println(result)
	// send data
	encoder.Encode(result)	
}
/*
	bool-chan uwu (◕‿◕✿)
*/
func runServer(done chan bool) {
	port := os.Getenv("PORT")
	if port == "" {
		fmt.Fprintf(os.Stderr, "PORT env variable not set\n")
		done<-true
	}
	fmt.Println("Running server, listening on port: ", port)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/list", listReports)	
	http.HandleFunc("/download", downloadFileHandler)	
	log.Fatal(http.ListenAndServe( ":" + port, nil))
	// send the done signal	
	done<- true
}

// use this function to run queries
func queryDb(query string) []Report {
	var rows pgx.Rows
	var err error
	reports := make([]Report, 0)	

	if prs := strings.Contains(query, "SELECT"); prs {
		// fmt.Println("SELECT query request recieved")	
		rows, err = dbpool.Query(context.Background(), query)
		if err != nil {
			fmt.Println("Error occured while querying: \n", err);	
			return nil	
		}
	} else {
		// create, insert, update kind of queries
		ct, err := dbpool.Exec(context.Background(), query)	
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occured while querying: %v\n", err);
			return nil 
		}
		fmt.Println("Command Tag returned: ", ct.String())
		return nil	
	} 

	// read return of query
	for r := 0; rows.Next(); r += 1 {
		// var result string
		values, err := rows.Values()
		
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occured while extracting values from rows: %v\n", err)	
			return reports	
		}	
		fmt.Println("iterating over row: ", r)
		var report Report	
		// fmt.Println(values)
		for i, value := range values {
			fmt.Fprintf(os.Stdout, "%v of type: %T, ", value, value);
			// fmt.Println("%v of type: %T", value, value)
			switch i {
				case 0:
					report.setId(int(value.(int32)))
				case 1:
					report.setSampleName(string(value.(string)))	
				case 2:
					report.setTestTimestamp(time.Time(value.(time.Time)))
				case 3, 4, 5, 6:
					report.setFeature(i - 2, string(value.(string)))	
				case 7:
					report.setFilePath(string(value.(string)))	
				case 8:
					report.setFinalVerdict(bool(value.(bool)))	
			}
		}	
		fmt.Println()
		reports = append(reports, report)	
	} 

	return reports 
}

func seedTableAndData(dbpool *pgxpool.Pool) {
	
	commandTag, err := dbpool.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS "Reports" (
			id SERIAL PRIMARY KEY,
			sample_name VARCHAR(20),
			test_datetime TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			feature_1 VARCHAR(50) NOT NULL,
			feature_2 VARCHAR(50),
			feature_3 VARCHAR(50),
			feature_4 VARCHAR(50),
			report_filepath TEXT UNIQUE,	
			final_verdict BOOLEAN NOT NULL 
		)`);
	
	fmt.Println("\nCREATE TABLE command tag returned: ", commandTag.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error returned while executing query: %v", err)
		os.Exit(1)	
	}
	
	// test the db/table dummy data	
	commandTag, err = dbpool.Exec(context.Background(), 
		`INSERT INTO "Reports"(id, sample_name, test_datetime, feature_1, feature_2, feature_3, feature_4, report_filepath, final_verdict) 
					  VALUES (DEFAULT, 'test', DEFAULT, 'test', 'test', 'test', 'test', '/tmp/test.txt', true)	
		`);

	fmt.Println("INSERT command tag returned: ", commandTag.String())	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error returned while executing query: %v", err)
		os.Exit(1)
	}	

	var rows pgx.Rows
	rows, err = dbpool.Query(context.Background(), "SELECT * FROM \"Reports\"")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error returned while executing query: %v\n", err)
		os.Exit(1)	
	}	
	
	fmt.Println("Result received after running seed:")
	// parse the rows into Record struct objects 	
	for ;rows.Next(); {
		values, err := rows.Values()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error returned while reading values from rows: %v\n", err)
			break	
		}
		// prints nothing 
		fmt.Println(values) 	
	}
}

func main() {
	// start server
	done := make(chan bool)
	go runServer(done)
	// run database driver, connect to db
	// URI := "postgresql://localhost:5432/postgres?user=postgres&password=1234	
	var err error	
	dbpool, err = pgxpool.New(context.Background(), URI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Ping successful: ", dbpool.Ping(context.Background()) == nil)
	
	// seedTableAndData(dbpool)	
	
	defer dbpool.Close()
	// defer terminating pool until the end of main

	// block until server returns
	<-done
}

