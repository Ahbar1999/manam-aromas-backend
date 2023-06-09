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
	"github.com/golang-jwt/jwt/v4"	
	"github.com/brianvoe/gofakeit/v6"
)

/*
	TODO:
*/


var URI string = "postgresql://localhost:5432/postgres?user=postgres&password=1234"
var dbpool *pgxpool.Pool
// could probably load this as an env variable 
var hmacSampleSecret []byte = []byte("secret_key")

func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		// 404- Bad Request
		w.WriteHeader(400)
		w.Write([]byte("make GET req with \"filepath\" parameter"))
		return
	}

	parsed, _ := url.Parse(r.RequestURI)
	q := parsed.Query()
	fn := filepath.FromSlash(strings.Join(q["filepath"], ""))
	fmt.Println("Sending file with filename: ")
	fmt.Println(fn)
	data, err := os.ReadFile(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v while reading filepath: %v\n", err, fn)  
	}
	
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(fn)))
	w.WriteHeader(http.StatusOK)
	// copy bytes from "data" stream to http "writer" 
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
	 	
	commandTag, err := dbpool.Exec(context.Background(), "DROP TABLE IF EXISTS \"Reports\"")	
	fmt.Println("\nDROP TABLE command tag returned: ", commandTag.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error returned while executing query: %v", err)
		os.Exit(1)	
	}	

	commandTag, err = dbpool.Exec(context.Background(),
		`CREATE TABLE "Reports" (
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

	// insert fake data into the db/table
	for i := 0; i < 25; i += 1 {
		var fakeReport Report 
		gofakeit.Struct(&fakeReport)		
		stmt := `INSERT INTO "Reports" (id, sample_name, test_datetime, feature_1, feature_2, feature_3, feature_4, report_filepath, final_verdict) 
						  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)	
			`

		_, err = dbpool.Exec(context.Background(), stmt, i, fakeReport.Sample_name, fakeReport.Test_datetime, fakeReport.Feature_1, fakeReport.Feature_2, fakeReport.Feature_3, fakeReport.Feature_4, "/tmp/" + fakeReport.Report_filepath + ".txt", fakeReport.Final_verdict);
	
		// fmt.Println("INSERT command tag returned: ", commandTag.String())	
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error returned while executing query: %v", err)
			os.Exit(1)
		}	
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

func authToken(tokenString string) error {	
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error){	
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// return hmacSampleSecret as the key to parse tokenString with
		return hmacSampleSecret, nil	
	})
	
	if !token.Valid {
		return err
	}	

	if claims, ok := token.Claims.(*CustomClaims); ok {
		if claims.IssuedAt == nil || claims.ExpiresAt == nil {		
			return fmt.Errorf("sorry, your auth token doesn't carry valid claims! \nPlease create a new one")	
		}	
		fmt.Printf("Authenticating %v \n", claims.Audience)
		fmt.Println("Bearing token issued at: ", claims.IssuedAt.String())
		//  check wether token expired or not
		// although this validation happens automatically somewhere while passing the token around it think 
		if ok := time.Now().Before(claims.ExpiresAt.Time); !ok {
			return fmt.Errorf("sorry, your auth token has expired \nPlease create a new one")	
		}	
	} else {
		return fmt.Errorf("sorry, couldnt parse claims from your auth token \nPlease create a new one")	
	}

	return nil 
}

type CustomClaims struct {
	Greetings string `json:"greetings"`
	jwt.RegisteredClaims
} 

func getNewToken(username string) string {
	// generate a tokee with claims
	claims := CustomClaims{
		"Hello User!",
		jwt.RegisteredClaims {
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// expires within 15 minute(s)
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			Issuer: "Admin",
			Audience: []string{username},	
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		
	// sign the token with hmac using out secret string
	tokenString, err := token.SignedString(hmacSampleSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "New token couldnt be created; ERROR: %v", err)
	}
	fmt.Println("New token created\n", tokenString, err)

	return tokenString
}

func checkCredentials(username, password string) bool {
	// create a Users table in the database
	// get users and authenticate against them 

	// this is temporary 
	return username == "ahbar" && password == "1234"	
}

func sendErrorResponse(w http.ResponseWriter, responseCode int, response []byte) {
	w.WriteHeader(responseCode)
	w.Write(response)
}

/*
	bool-chan uwu (◕‿◕✿)
*/
func middleWare(w http.ResponseWriter, r *http.Request) {		
	// create new token upon request 	
	if r.URL.Path == "/auth" {
		err := r.ParseForm()	
		if err != nil {
			sendErrorResponse(w, http.StatusBadRequest, []byte("Authentication details not provided!\n Send a post request with form values for username and password\n"))	
			return
		}	
		// check for token or credentials
		fmt.Println(r.Form)
		if ok := checkCredentials(r.Form.Get("username"), r.Form.Get("password")); !ok {
			sendErrorResponse(w, http.StatusForbidden, []byte("Authentication failed!"))		
			return 
		}
		
		// generate new token and return it for client use
		newTokenString := struct { 
			TokenString string `json:"token_string"`
		}{
			getNewToken(r.Form.Get("username")),
		}
		// encode it as json and send it
		encoder := json.NewEncoder(w)
		w.WriteHeader(http.StatusOK)
		// now on the recieving side token can be accessed with token_string key
		encoder.Encode(newTokenString)
		return
	}
	
	// otherwise authenticate token
	tokenString := r.Header.Get("Authorization")
	
	if tokenString == "" {
		sendErrorResponse(w, http.StatusBadRequest, []byte("jwt Auth token not provided"))	
		return	
	}
	
	// fmt.Printf("Recieved token string %v\n", tokenString)
	if err := authToken(tokenString); err != nil {
		sendErrorResponse(w, http.StatusForbidden, []byte(err.Error()))
		return	
	}

	// authenticated
	// continue processing the req
	// fmt.Println("Resource requested on: ", r.URL.Path)
	switch r.URL.Path {
	case "/":
		indexHandler(w, r)
	case "/upload":
		uploadHandler(w, r)
	case "/list":
		listReports(w, r)
	case "/download":
		downloadFileHandler(w, r)
	default:
		sendErrorResponse(w, http.StatusNotFound, []byte("The resource you requested does not exist"))
	}
}

func runServer(done chan bool) {
	port := os.Getenv("PORT")
	if port == "" {
		fmt.Fprintf(os.Stderr, "PORT env variable not set\n")
		done<-true
	}

	fmt.Println("Running server, listening on port: ", port)
	
	// route all requests to middleWare	
	http.HandleFunc("/", middleWare)
	
	log.Fatal(http.ListenAndServe( ":" + port, nil))
	// send the done signal	
	done<- true
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
	
	seedTableAndData(dbpool)	
	
	defer dbpool.Close()
	// defer terminating pool until the end of main

	// block until server returns
	<-done
}

