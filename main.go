package main

import (
	"encoding/json"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"io/ioutil"
	"math"
	"math/rand"
	"strconv"
)

const ExcelSheetLocation = "/Users/johmagnu/Desktop/scripts/MarysvillePilchuckStudentSplit/MPHSClassSchedules.xlsx"
const SheetName = "Sheet1"
const MaxIterationsFromRandomStartingPoint = 500
const MaxStartingPoints = 2000

var ColumnsWithClasses = []int{6, 8, 10, 12, 14, 16}

type Student struct {
	Id      int      `json:"Id"`
	Name    string   `json:"Name"`
	Classes []string `json:"Classes"`
}

type BestSplit struct {
	GroupA []Student `json:"GroupA"`
	GroupB []Student `json:"GroupB"`
}

type GroupStats struct {
	Score                      int            `json:"Score"`
	TotalClassesBetween16And18 int            `json:"TotalClassesBetween16And18"`
	TotalClassesBetween18And20 int            `json:"TotalClassesBetween18And20"`
	TotalClassesLargerThan20   int            `json:"TotalClassesLargerThan20"`
	MaxClassSize               int            `json:"MaxClassSize"`
	StudentIdToScoreMap        map[int]int    `json:"StudentIdToScoreMap"`
	ClassToSizeMap             map[string]int `json:"ClassToSizeMap"`
}

type BestSplitStats struct {
	GroupA GroupStats `json:"GroupA"`
	GroupB GroupStats `json:"GroupB"`
}

type ClassDistribution struct {
	TotalClasses int `json:"TotalClasses"`
	TotalStudents int `json:"TotalStudents"`
	ClassesToSizeMap map[string]int `json:"ClassesToSizeMap"`
}

func main() {
	students, err := getStudentsFromExcelSheet()
	if err != nil {
		fmt.Println(err)
		return
	}
	saveClassDistribution(students)
	studentIdToStudentMap := map[int]Student{}
	for _, student := range students {
		studentIdToStudentMap[student.Id] = student
	}

	bestScore := math.MaxInt32
	bestSplit := BestSplit{}
	bestSplitStats := BestSplitStats{}

	for j := 0; j < MaxStartingPoints; j++ {
		groupA, groupB := getRandomSplit(students)
		for i := 0; i < MaxIterationsFromRandomStartingPoint; i++ {
			groupAStats := getStatsForGroup(groupA)
			groupBStats := getStatsForGroup(groupB)

			// Check if split is best seen so far and print statistics
			if groupAStats.Score+groupBStats.Score < bestScore {
				bestSplit = BestSplit{
					GroupA: groupA,
					GroupB: groupB,
				}
				bestSplitStats = BestSplitStats{
					GroupA: groupAStats,
					GroupB: groupBStats,
				}
				bestScore = groupAStats.Score+groupBStats.Score

				fmt.Printf("found better: %v on iteration: %d\n", groupAStats.Score+groupBStats.Score, i)

				// save best to file so we have record in case program crashes
				bestSplitJson, _ := json.Marshal(bestSplit)
				err = ioutil.WriteFile("bestSplit.json", bestSplitJson, 0644)

				bestSplitStatsJson, _ := json.Marshal(bestSplitStats)
				err = ioutil.WriteFile("bestSplitStats.json", bestSplitStatsJson, 0644)
			}

			// Compute new groups
			groupA, groupB = getNewGroups(groupAStats.StudentIdToScoreMap, groupBStats.StudentIdToScoreMap, studentIdToStudentMap)
		}
 	}
}

func getNewGroups(studentIdToScoreMapA, studentIdToScoreMapB map[int]int, studentIdToStudentMap map[int]Student) ([]Student, []Student) {
	var groupANew []Student
	var groupBNew []Student
	flippedStudents := 0
	for studentId, score := range studentIdToScoreMapA {
		// flip group with probability proportional to how many Classes the student was in that were too large
		if score == 0 || rand.Intn(70) > rand.Intn(score) {
			groupANew = append(groupANew, studentIdToStudentMap[studentId])
		} else {
			flippedStudents += 1
			groupBNew = append(groupBNew, studentIdToStudentMap[studentId])
		}
	}
	for studentId, score := range studentIdToScoreMapB {
		// flip group with probability proportional to how many Classes the student was in that were too large
		if score == 0 || rand.Intn(70) > rand.Intn(score) {
			groupBNew = append(groupBNew, studentIdToStudentMap[studentId])
		} else {
			flippedStudents += 1
			groupANew = append(groupANew, studentIdToStudentMap[studentId])
		}
	}

	return groupANew, groupBNew
}

func getStatsForGroup(group []Student) GroupStats {
	classesToStudentIdMap := map[string][]int{}
	for _, student := range group {
		for _, class := range student.Classes {
			if studentIds, ok := classesToStudentIdMap[class]; ok {
				studentIds = append(studentIds, student.Id)
				classesToStudentIdMap[class] = studentIds
			} else {
				classesToStudentIdMap[class] = []int{student.Id}
			}
		}
	}

	score := 0
	totalClassesBetween16And18 := 0
	totalClassesBetween18And20 := 0
	totalClassesLargerThan20 := 0
	maxClassSize := 0
	studentIdToScoreMap := map[int]int{}
	for _, studentIds := range classesToStudentIdMap {
		if len(studentIds) > maxClassSize {
			maxClassSize = len(studentIds)
		}

		if len(studentIds) >= 20 {
			score += 7
			totalClassesLargerThan20 += 1
			for _, studentId := range studentIds {
				if _, ok := studentIdToScoreMap[studentId]; ok {
					studentIdToScoreMap[studentId] += 7
				} else {
					studentIdToScoreMap[studentId] = 7
				}
			}
		} else if len(studentIds) >= 18 {
			score += 3
			totalClassesBetween18And20 += 1
			for _, studentId := range studentIds {
				if _, ok := studentIdToScoreMap[studentId]; ok {
					studentIdToScoreMap[studentId] += 3
				} else {
					studentIdToScoreMap[studentId] = 3
				}
			}
		} else if len(studentIds) >= 16 {
			score += 1
			totalClassesBetween16And18 += 1
			for _, studentId := range studentIds {
				if _, ok := studentIdToScoreMap[studentId]; ok {
					studentIdToScoreMap[studentId] += 1
				} else {
					studentIdToScoreMap[studentId] = 1
				}
			}
		} else {
			for _, studentId := range studentIds {
				if _, ok := studentIdToScoreMap[studentId]; !ok {
					studentIdToScoreMap[studentId] = 0
				}
			}
		}
	}

	classesToSizeMap := map[string]int{}
	for class, studentIds := range classesToStudentIdMap {
		classesToSizeMap[class] = len(studentIds)
	}

	return GroupStats{
		Score:                      score,
		TotalClassesBetween16And18: totalClassesBetween16And18,
		TotalClassesBetween18And20: totalClassesBetween18And20,
		TotalClassesLargerThan20:   totalClassesLargerThan20,
		MaxClassSize:               maxClassSize,
		StudentIdToScoreMap:        studentIdToScoreMap,
		ClassToSizeMap:             classesToSizeMap,
	}
}


func saveClassDistribution(students []Student) {
	classesToStudentIdMap := map[string][]int{}
	for _, student := range students {
		for _, class := range student.Classes {
			if studentIds, ok := classesToStudentIdMap[class]; ok {
				studentIds = append(studentIds, student.Id)
				classesToStudentIdMap[class] = studentIds
			} else {
				classesToStudentIdMap[class] = []int{student.Id}
			}
		}
	}

	classesToSizeMap := map[string]int{}
	for class, studentIds := range classesToStudentIdMap {
		classesToSizeMap[class] = len(studentIds)
	}

	studentIdsSet := map[int]int{}
	for _, studentIds := range classesToStudentIdMap {
		for _, studentId := range studentIds {
			if _, ok := studentIdsSet[studentId]; !ok {
				studentIdsSet[studentId] = 1
			}
		}
	}

	classDistribution := ClassDistribution{
		TotalClasses:     len(classesToStudentIdMap),
		TotalStudents:    len(studentIdsSet),
		ClassesToSizeMap: classesToSizeMap,
	}

	classDistributionJson, _ := json.Marshal(classDistribution)
	err := ioutil.WriteFile("classDistribution.json", classDistributionJson, 0644)
	if err != nil {
		fmt.Printf("Error while writing class distribution to file: %v", err)
	}

	f := excelize.NewFile()
	rowNum := 1
	for class, size := range classDistribution.ClassesToSizeMap {
		f.SetCellValue("Sheet1", "A" + strconv.Itoa(rowNum), class)
		f.SetCellValue("Sheet1", "B" + strconv.Itoa(rowNum), size)
		rowNum++
	}
	f.SaveAs("./ClassDistribution.xlsx")
}

func getRandomSplit(students []Student) ([]Student, []Student) {
	var groupA []Student
	var groupB []Student
	for _, student := range students {
		if rand.Intn(2) == 0 {
			groupA = append(groupA, student)
		} else {
			groupB = append(groupB, student)
		}
	}
	return groupA, groupB
}

func getStudentsFromExcelSheet() ([]Student, error) {
	xlsx, err := excelize.OpenFile(ExcelSheetLocation)
	if err != nil {
		return nil, err
	}
	rows, err := xlsx.GetRows(SheetName)
	if err != nil {
		return nil, err
	}

	var students []Student
	for idx, row := range rows {
		if idx != 0 {
			var classes []string
			for _, col := range ColumnsWithClasses {
				if len(row) > col && row[col] != "" {
					classes = append(classes, row[col]) // + "_" + strconv.Itoa((col - 4) / 2))
				}
			}
			students = append(students, Student{
				Id:      idx,
				Name:    row[0],
				Classes: classes,
			})
		}
	}
	return students, nil
}
