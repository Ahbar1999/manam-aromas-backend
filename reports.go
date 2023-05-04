package main

import (
	"time"
	"os"
	"path/filepath"
)

type Report struct {
	Id 				int			`json:"id"`  	
	Sample_name 	string		`json:"sample_name"`  
	Test_datetime 	time.Time	`json:"test_datetime"` 
	Feature_1 		string		`json:"feature_1"` 			 	
	Feature_2 		string		`json:"feature_2"`
	Feature_3 		string		`json:"feature_3"`
	Feature_4 		string		`json:"feature_4"`
	Report_filepath string		`json:"report"`
	Final_verdict 	bool		`json:"final_verdict"`
}

func (report *Report) setId(id int) {
	report.Id = id
} 

func (report *Report) setSampleName(name string) {
	report.Sample_name = name 
} 

func (report *Report) setTestTimestamp(timestamp time.Time) {
	report.Test_datetime = timestamp 
} 

func (report *Report) setFeature(id int, value string) {
	switch id {
		case 1:
			report.Feature_1 = value
		case 2:
			report.Feature_2 = value
		case 3:
			report.Feature_3 = value
		case 4:
			report.Feature_4 = value
	}
}

func (report *Report) setFilePath(path string) {
	cwd, _ := os.Getwd()
	// filepath.FromSlash helps maintain platform compatible path syntax
	report.Report_filepath = cwd + filepath.FromSlash(path) 
}

func (report *Report) setFinalVerdict(verdict bool) {
	report.Final_verdict = verdict
}
