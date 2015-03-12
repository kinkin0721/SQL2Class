// SQL2Class project main.go
package main

import (
	"SQL2Class/MapSorter"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"sort"
	"strconv"
	"strings"
)

func checkError(err error) {
	if err != nil {
		//fmt.Println(err)
		panic(err.Error())
	}
}

func get_column_infos(column_type string) (type_name string, type_info string) {

	canadd := true
	var size_start int
	var size_end int

	for i := 0; i < len(column_type); i++ {

		s := column_type[i : i+1]

		if s == "(" {
			canadd = false
			size_start = i + 1
		} else if s == ")" {
			canadd = true
			size_end = i
			continue
		}

		if canadd {
			type_name += s
		}
	}

	type_info = column_type[size_start:size_end]

	return
}

func make_type(column_type string, column_size string) string {

	if column_type != "enum" {
		return make_type_common(column_type, column_size)
	} else {
		return make_type_enum(column_type, column_size)
	}
}

func make_type_common(column_type string, column_size string) string {

	var size int

	if column_size != "" && column_type != "enum" { //float and enum
		var err error
		size, err = strconv.Atoi(column_size)
		if err != nil {
			fmt.Println(column_type)
			panic(err)
		}
	} else {
		size = 0
	}

	if column_type == "tinyint" && size == 1 {
		return "bool"
	} else if column_type == "bigint" {
		return "int64"
	} else if column_type == "bigint unsigned" {
		return "uint64"
	} else if column_type == "int" {
		return "int32"
	} else if column_type == "int unsigned" {
		return "uint32"
	} else if column_type == "smallint" {
		return "int16"
	} else if column_type == "smallint unsigned" {
		return "uint16"
	} else if column_type == "tinyint" {
		return "int8"
	} else if column_type == "tinyint unsigned" {
		return "uint8"
	} else if column_type == "float" {
		return "float32"
	} else if column_type == "enum" {
		return "byte"
	} else if column_type == "varchar" || column_type == "text" || column_type == "blob" || column_type == "date" || column_type == "datetime" {
		return "string"
	} else {
		return column_type
	}
}

func make_type_enum(column_type string, enum_info string) string {
	return "byte //" + enum_info
}

func make_bd_template(dataSourceName string, sql_base string) {

	db, err := sql.Open("mysql", dataSourceName)
	checkError(err)
	defer db.Close()

	rows, err := db.Query("select table_name,column_type,column_name from INFORMATION_SCHEMA.columns where table_schema = '" + sql_base + "'")
	checkError(err)
	defer rows.Close()

	struct_string := make(map[string]string)
	map_string := make(map[string]string)
	mapinit_string := make(map[string]string)

	var temp_column_type_name, table_name, column_name string
	for rows.Next() {
		err := rows.Scan(&table_name, &temp_column_type_name, &column_name)
		checkError(err)

		table_name = strings.ToUpper(table_name[0:1]) + table_name[1:]

		_, ok := struct_string[table_name]
		if !ok {
			struct_string[table_name] = "type " + table_name + " struct {\n"
		}

		column_type_name, column_type_size := get_column_infos(temp_column_type_name)
		column_type := make_type(column_type_name, column_type_size)
		column_type_common := make_type_common(column_type_name, column_type_size)
		if column_name == "type" {
			column_name = "_type"
		}
		struct_string[table_name] += "\t" + column_name + " " + column_type + "\n"

		_, ok = mapinit_string[table_name]
		if !ok {
			map_string[table_name] = "var " + table_name + "s map[" + column_type_common + "]DBStruct." + table_name + "\n"
			mapinit_string[table_name] = "\t" + table_name + "s := make(map[" + column_type_common + "]DBStruct." + table_name + ")\n\t_ = " + table_name + "s\n"
		}
	}

	mss := MapSorter.NewMapSorter(struct_string)
	mms := MapSorter.NewMapSorter(map_string)
	mmis := MapSorter.NewMapSorter(mapinit_string)
	sort.Sort(mss)
	sort.Sort(mms)
	sort.Sort(mmis)

	var final_struct_string string = "package DBStruct\n\n"
	var final_map_string string = "package DB\n\nimport (\n\t\"testMysql/DB/struct\"\n)\n\n"
	var final_mapinit_string string = "func init() {\n"

	for _, val := range mss {
		val.Val += "}\n\n"
		final_struct_string += val.Val
	}

	for _, val := range mms {
		final_map_string += val.Val
	}

	for _, val := range mmis {
		final_mapinit_string += val.Val
	}
	final_mapinit_string += "}"

	err = os.Remove("temp\\db_templaet.go")
	if err != nil {
		fmt.Println(err)
	}

	file, err := os.Create("temp\\db_templaet.go")
	checkError(err)
	file.Close()

	file, err = os.OpenFile("temp\\db_templaet.go", os.O_APPEND, 777)
	checkError(err)
	defer file.Close()

	_, err = file.WriteString(final_struct_string)
	checkError(err)

	err = os.Remove("temp\\DBTemplateLoader.go")
	if err != nil {
		fmt.Println(err)
	}

	file_loader, err := os.Create("temp\\DBTemplateLoader.go")
	checkError(err)
	file.Close()

	file_loader, err = os.OpenFile("temp\\DBTemplateLoader.go", os.O_APPEND, 777)
	checkError(err)
	defer file_loader.Close()

	_, err = file_loader.WriteString(final_map_string + "\n" + final_mapinit_string)
	checkError(err)
}

func main() {

	sql_ip_port := "172.26.48.2:3306"
	sql_user := "root"
	sql_pwd := "root"

	sql_base := "js_base"
	//sql_game := "js_test_s0"
	//sql_cross := "js_crossserver_s0"

	arg_num := len(os.Args)

	if arg_num > 1 {
		sql_ip_port = os.Args[1]
	}
	if arg_num > 2 {
		sql_user = os.Args[2]
	}
	if arg_num > 3 {
		sql_pwd = os.Args[3]
	}
	if arg_num > 4 {
		sql_base = os.Args[4]
	}
	if arg_num > 5 {
		//sql_game = os.Args[5]
	}
	if arg_num > 6 {
		//sql_cross = os.Args[6]
	}

	make_bd_template(sql_user+":"+sql_pwd+"@tcp("+sql_ip_port+")/js_base", sql_base)
}
