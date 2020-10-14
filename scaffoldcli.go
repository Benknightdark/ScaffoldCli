package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/elgs/gosqljson"
	"github.com/urfave/cli/v2"
	"html/template"
	"log"
	"os"
	"strconv"
	"sync"
)

type ModelScaffold struct {
	ModelPath   string
	Module      string
	ImportUUID  string
	TableName   string
	TableSchema string
	ColData     []ColData
}
type ColData struct {
	ColName    string
	DataType   string
	Annotation string
}

func createfolder() {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(path, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

	}
}
func createFile(fileName string, struct_content string) {
	// check if file exists
	var _, err = os.Stat(path + "/" + fileName + ".go")

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path + "/" + fileName + ".go")
		if isError(err) {
			return
		}
		defer file.Close()
	}

	fmt.Println("File Created Successfully", path+"/"+fileName+".go")
	writeFile(path+"/"+fileName+".go", struct_content)
}

func writeFile(fileName string, struct_content string) {
	// Open file using READ & WRITE permission.
	var file, err = os.OpenFile(fileName, os.O_RDWR, 0644)
	if isError(err) {
		return
	}
	defer file.Close()

	// Write some text line-by-line to file.
	_, err = file.WriteString(struct_content)
	if isError(err) {
		return
	}

	// Save file changes.
	err = file.Sync()
	if isError(err) {
		return
	}

	fmt.Println("File Updated Successfully. => " + fileName)
}

func deleteFile() {
	err := os.RemoveAll(path)
	if err != nil {
		log.Fatal(err)
	}
}

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}

	return (err != nil)
}

//  執行sql查詢指令，並回傳結果
func exec_sql(sql_string string) []map[string]string {
	runSql := fmt.Sprintf(sql_string)
	nPort, err2 := strconv.ParseInt(port, 10, 64)
	if err2 != nil {
		log.Fatal("Error creating connection pool: ", err2.Error())

	}
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, nPort, database)

	var err error

	// Create connection pool
	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatal("Error creating connection pool: ", err.Error())
	}
	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}
	rows, err := db.Query(runSql)
	if err != nil {
		panic(err.Error())
	}
	theCase := "" // "lower", "upper", "camel" or the orignal case if this is anything other than these three

	data2, _ := gosqljson.QueryDbToMap(db, theCase, runSql)
	defer rows.Close()
	//defer db.Close()
	//fmt.Println(data2[0]["TABLE_CATALOG"])
	return data2
}
func create_table_struct() {
	// logger := log.New(os.Stdout, "", 0)

	wg := new(sync.WaitGroup)
	deleteFile()
	createfolder()
	table_data := exec_sql(`SELECT 
				*
			FROM
				information_schema.tables
			WHERE TABLE_TYPE=N'BASE TABLE' AND TABLE_SCHEMA !=N'others'`)

	for val := range table_data {
		wg.Add(val)
		go func(table_info map[string]string) {
			var pa *ModelScaffold
			pa = new(ModelScaffold)
			pa.ModelPath = path
			pa.Module = module
			pa.TableName = table_info["TABLE_NAME"]
			pa.TableSchema = table_info["TABLE_SCHEMA"]
			// t := template.Must(template.ParseGlob("templates/*.tmpl"))
			// template.Must(t.ParseGlob("templates/db-model.tmpl"))

			t := template.Must(template.New("").Parse(`package {{.Module}} 		
			import (
					"gorm.io/gorm"
					{{if .ImportUUID -}}
					"{{.ImportUUID}}"
					{{- end}}
			)
			type {{.TableName}} struct {
				{{range .ColData -}}
					{{ .ColName }}  {{ .DataType }}  {{.Annotation -}}
				{{- end}}				
			}
			// 取得table (func名稱設成首字大寫，才能讓外部程式使用)
			func Get{{.TableName}}() func(db *gorm.DB) *gorm.DB {
				return func(db *gorm.DB) *gorm.DB {
					return db.Table("{{.TableSchema}}.{{.TableName}}")
					}
			}`))
			//time.Sleep(100 * time.Millisecond)

			println(table_info["TABLE_NAME"])
			col_data := exec_sql(fmt.Sprintf(`SELECT *
				FROM INFORMATION_SCHEMA.COLUMNS
				WHERE TABLE_SCHEMA=N'%s' AND TABLE_NAME = N'%s'`, table_info["TABLE_SCHEMA"], table_info["TABLE_NAME"]))
			import_uuid := ""
			for col := range col_data {
				data_type := "string"

				if col_data[col]["DATA_TYPE"] == "uniqueidentifier" {
					data_type = "uuid.UUID"
					import_uuid = `github.com/google/uuid`
				}
				append_col_data := ColData{ColName: col_data[col]["COLUMN_NAME"], DataType: data_type,
					Annotation: "`" + col_data[col]["DATA_TYPE"] + "`" + "\n"}
				pa.ColData = append(pa.ColData, append_col_data)
			}
			pa.ImportUUID = import_uuid
			var tpl bytes.Buffer
			err := t.Execute(&tpl, pa)
			if err != nil {
				panic(err)
			}
			print(tpl.String())
			createFile(col_data[0]["TABLE_NAME"], tpl.String())
			defer wg.Done()
		}(table_data[val])
	}
	wg.Wait()
	fmt.Println("==========================done==========================")
	// time.Sleep(5 * time.Second)

}

var path string
var module string
var server string
var port string
var user string
var password string
var database string
var db *sql.DB

func main() {

	app := &cli.App{
		Name:  "ScaffoldCli",
		Usage: "透過cli將mssql的table轉成golang的struct",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "path",
				Value:       "",
				Aliases:     []string{"p"},
				Usage:       "檔案儲存路徑",
				Destination: &path,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "module",
				Value:       "",
				Aliases:     []string{"m"},
				Usage:       "模組名稱",
				Destination: &module,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "server",
				Value:       "",
				Aliases:     []string{"s"},
				Usage:       "資料庫伺服器名稱",
				Destination: &server,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "port",
				Value:       "1433",
				Aliases:     []string{"po"},
				Usage:       "Port Number",
				Destination: &port,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "user",
				Value:       "",
				Aliases:     []string{"u"},
				Usage:       "登入帳號",
				Destination: &user,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "password",
				Value:       "",
				Aliases:     []string{"pa"},
				Usage:       "登入密碼",
				Destination: &password,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "database",
				Value:       "",
				Aliases:     []string{"d"},
				Usage:       "資料庫名稱",
				Destination: &database,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			create_table_struct()
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {

		log.Fatal(err)

	}
	os.Exit(0)

}
