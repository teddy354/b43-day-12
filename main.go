package main

import (
	"context"
	"fmt"
	"strings"
	"table-relation/connection"
	"table-relation/middleware"

	"log"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	route := mux.NewRouter()

	connection.DatabaseConnection() //connect to database

	//to access public folder
	route.PathPrefix("/public/").Handler(http.StripPrefix("/public", http.FileServer(http.Dir("./public/"))))
	route.PathPrefix("/upload/").Handler(http.StripPrefix("/upload/", http.FileServer(http.Dir("./upload/"))))

	route.HandleFunc("/", homePage).Methods("GET")
	route.HandleFunc("/contact", contactPage).Methods("GET")

	//Create
	route.HandleFunc("/project", projectPage).Methods("GET")
	route.HandleFunc("/project", middleware.UploadFile(addProject)).Methods("POST")
	//Read
	route.HandleFunc("/project/{id}", detailProject).Methods("GET")
	//Update
	route.HandleFunc("/editProject/{id}", editProject).Methods("GET")
	route.HandleFunc("/updateProject/{id}", middleware.UploadFile(updateProject)).Methods("POST")
	//Delete
	route.HandleFunc("/deleteProject/{id}", deleteProject).Methods("GET")

	//Register
	route.HandleFunc("/register", registerPage).Methods("GET")
	route.HandleFunc("/register", register).Methods("POST")
	//Login
	route.HandleFunc("/login", loginPage).Methods("GET")
	route.HandleFunc("/login", login).Methods("POST")
	//Logout
	route.HandleFunc("/logout", logout).Methods("GET")

	fmt.Println("Server running on port:8000")
	http.ListenAndServe("localhost:8000", route)
}

// // Struct buat menentukan variable sama tipe data nya
type dataReceive struct {
	ID           int
	Projectname  string
	Description  string
	Technologies []string
	Startdate    time.Time
	Enddate      time.Time
	Duration     string
	Image        string
	User         int
}

type User struct {
	ID       int
	Name     string
	Email    string
	Password string
}

type MetaData struct {
	ID        int
	IsLogin   bool
	UserName  string
	FlashData string
}

var Data = MetaData{}

func homePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("view/index.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	var result []dataReceive

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	if session.Values["IsLogin"] != true {
		Data.IsLogin = false

		rows, errQuery := connection.Conn.Query(context.Background(), "SELECT project_name, start_date, end_date, description, technologies, image FROM tb_projects")
		if errQuery != nil {
			fmt.Println("Message : " + errQuery.Error())
			return
		}

		for rows.Next() {
			var each = dataReceive{}

			err := rows.Scan(&each.Projectname, &each.Startdate, &each.Enddate, &each.Description, &each.Technologies, &each.Image)
			if err != nil {
				fmt.Println("Message : " + err.Error())
				return
			}

			each.Duration = countduration(each.Startdate, each.Enddate)

			result = append(result, each)
		}
	} else {
		Data.IsLogin = session.Values["IsLogin"].(bool)
		Data.UserName = session.Values["Name"].(string)

		user := session.Values["Id"]

		rows, errQuery := connection.Conn.Query(context.Background(), "SELECT tb_projects.id, project_name, start_date, end_date, description, technologies, image FROM tb_user LEFT JOIN tb_projects ON tb_projects.user_id = tb_user.id WHERE tb_projects.user_id = $1", user)
		if errQuery != nil {
			fmt.Println("Message : " + errQuery.Error())
			return
		}

		for rows.Next() {
			var each = dataReceive{}

			err := rows.Scan(&each.ID, &each.Projectname, &each.Startdate, &each.Enddate, &each.Description, &each.Technologies, &each.Image)
			if err != nil {
				fmt.Println("Message : " + err.Error())
				return
			}

			each.Duration = countduration(each.Startdate, each.Enddate)

			result = append(result, each)
		}
	}

	fm := session.Flashes("message")

	var flashes []string

	if len(fm) > 0 {
		session.Save(r, w)

		for _, fl := range fm {
			flashes = append(flashes, fl.(string))
		}
	}

	Data.FlashData = strings.Join(flashes, "")

	dataCaller := map[string]interface{}{
		"Projects": result,
		"Data":     Data,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, dataCaller)
}

func projectPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("view/myProject.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	if session.Values["IsLogin"] != true {
		Data.IsLogin = false
	} else {
		Data.IsLogin = session.Values["IsLogin"].(bool)
		Data.UserName = session.Values["Name"].(string)
	}

	dataCaller := map[string]interface{}{
		"Data": Data,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, dataCaller)
}

func countduration(start time.Time, end time.Time) string {
	distance := end.Sub(start)

	//Ubah milisecond menjadi bulan, minggu dan hari
	monthDistance := int(distance.Hours() / 24 / 30)
	weekDistance := int(distance.Hours() / 24 / 7)
	daysDistance := int(distance.Hours() / 24)

	// variable buat menampung durasi yang sudah diolah
	var duration string
	// pengkondisian yang akan mengirimkan durasi yang sudah diolah
	if monthDistance >= 1 {
		duration = strconv.Itoa(monthDistance) + " months"
	} else if monthDistance < 1 && weekDistance >= 1 {
		duration = strconv.Itoa(weekDistance) + " weeks"
	} else if monthDistance < 1 && daysDistance >= 0 {
		duration = strconv.Itoa(daysDistance) + " days"
	} else {
		duration = "0 days"
	}
	// Duration End

	return duration
}

func addProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10485760)

	if err != nil {
		log.Fatal(err)
	}

	projectname := r.PostForm.Get("project-name")
	description := r.PostForm.Get("description")
	technologies := r.Form["technologies"]

	const timeFormat = "2006-01-02"
	startDate, _ := time.Parse(timeFormat, r.PostForm.Get("start-date"))
	endDate, _ := time.Parse(timeFormat, r.PostForm.Get("end-date"))

	dataContex := r.Context().Value("dataFile")
	image := dataContex.(string)

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")
	user := session.Values["Id"].(int)

	_, insertRow := connection.Conn.Exec(context.Background(), "INSERT INTO tb_projects(project_name, start_date, end_date, description, technologies, image, user_id) VALUES ($1, $2, $3, $4, $5, $6, $7)", projectname, startDate, endDate, description, technologies, image, user)
	if insertRow != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message: " + err.Error()))
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func detailProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("view/project-detail.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	ID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var resultData = dataReceive{}

	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM tb_projects WHERE id=$1", ID).Scan(&resultData.ID, &resultData.Projectname, &resultData.Startdate, &resultData.Enddate, &resultData.Description, &resultData.Technologies, &resultData.Image, &resultData.User)

	resultData.Duration = countduration(resultData.Startdate, resultData.Enddate)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	if session.Values["IsLogin"] != true {
		Data.IsLogin = false
	} else {
		Data.IsLogin = session.Values["IsLogin"].(bool)
		Data.UserName = session.Values["Name"].(string)
	}

	detailProject := map[string]interface{}{
		"Projects": resultData,
		"Data":     Data,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, detailProject)
}

func deleteProject(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	_, deleteRows := connection.Conn.Exec(context.Background(), "DELETE FROM tb_projects WHERE id=$1", id)
	if deleteRows != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + deleteRows.Error()))
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func editProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("view/editProject.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	if session.Values["IsLogin"] != true {
		Data.IsLogin = false
	} else {
		Data.IsLogin = session.Values["IsLogin"].(bool)
		Data.UserName = session.Values["Name"].(string)
	}

	ID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var editData = dataReceive{}

	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM tb_projects WHERE id=$1", ID).Scan(&editData.ID, &editData.Projectname, &editData.Startdate, &editData.Enddate, &editData.Description, &editData.Technologies, &editData.Image, &editData.User)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	dataEdit := map[string]interface{}{
		"Projects": editData,
		"Data":     Data,
	}

	tmpl.Execute(w, dataEdit)
}

func updateProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10485760)

	if err != nil {
		log.Fatal(err)
	}

	projectname := r.PostForm.Get("project-name")
	description := r.PostForm.Get("description")
	technologies := r.Form["technologies"]

	const timeFormat = "2006-01-02"
	startDate, _ := time.Parse(timeFormat, r.PostForm.Get("start-date"))
	endDate, _ := time.Parse(timeFormat, r.PostForm.Get("end-date"))
	ID, _ := strconv.Atoi(mux.Vars(r)["id"])

	dataContex := r.Context().Value("dataFile")
	image := dataContex.(string)

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")
	author := session.Values["Id"].(int)

	_, updateRow := connection.Conn.Exec(context.Background(), "UPDATE tb_projects SET project_name = $1, start_date = $2, end_date = $3, description = $4, technologies = $5, image = $6, user_id = $7 WHERE id = $8", projectname, startDate, endDate, description, technologies, image, author, ID)
	if updateRow != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message: " + err.Error()))
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func contactPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("view/contact.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	if session.Values["IsLogin"] != true {
		Data.IsLogin = false
	} else {
		Data.IsLogin = session.Values["IsLogin"].(bool)
		Data.UserName = session.Values["Name"].(string)
	}

	dataCaller := map[string]interface{}{
		"Data": Data,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, dataCaller)
}

func registerPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("view/register.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	if session.Values["IsLogin"] != true {
		Data.IsLogin = false
	} else {
		Data.IsLogin = session.Values["IsLogin"].(bool)
		Data.UserName = session.Values["Name"].(string)
	}

	dataCaller := map[string]interface{}{
		"Data": Data,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, dataCaller)
}

func register(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	name := r.PostForm.Get("name")
	email := r.PostForm.Get("email")

	password := r.PostForm.Get("password")
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	_, insertRow := connection.Conn.Exec(context.Background(), "INSERT INTO tb_user(name, email, password) VALUES ($1, $2, $3)", name, email, passwordHash)
	if insertRow != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + insertRow.Error()))
		return
	}

	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("view/login.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	if session.Values["IsLogin"] != true {
		Data.IsLogin = false
	} else {
		Data.IsLogin = session.Values["IsLogin"].(bool)
		Data.UserName = session.Values["Name"].(string)
	}

	dataCaller := map[string]interface{}{
		"Data": Data,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, dataCaller)
}

func login(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	email := r.PostForm.Get("email")
	password := r.PostForm.Get("password")

	user := User{}

	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM tb_user WHERE email = $1", email).Scan(&user.ID, &user.Name, &user.Email, &user.Password)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	session.Values["IsLogin"] = true
	session.Values["Name"] = user.Name
	session.Values["Id"] = user.ID
	session.Options.MaxAge = 10800

	session.AddFlash("Login success", "message")
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func logout(w http.ResponseWriter, r *http.Request) {
	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	session.Options.MaxAge = -1

	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
