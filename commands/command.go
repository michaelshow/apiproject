package commands

import (
	"apiproject/conf"
	"apiproject/models"
	"apiproject/utils/filetil"
	//"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	//"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/i18n"
	"github.com/lifei6671/gocaptcha"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RegisterDataBase 注册数据库
func RegisterDataBase() {
	logs.Info("正在初始化数据库配置.")
	dbadapter := beego.AppConfig.String("db_adapter")
	orm.DefaultTimeLoc = time.Local
	orm.DefaultRowsLimit = -1

	if strings.EqualFold(dbadapter, "mysql") {
		host := beego.AppConfig.String("db_host")
		database := beego.AppConfig.String("db_database")
		username := beego.AppConfig.String("db_username")
		password := beego.AppConfig.String("db_password")

		timezone := beego.AppConfig.String("timezone")
		location, err := time.LoadLocation(timezone)
		if err == nil {
			orm.DefaultTimeLoc = location
		} else {
			logs.Error("加载时区配置信息失败,请检查是否存在 ZONEINFO 环境变量->", err)
		}

		port := beego.AppConfig.String("db_port")

		dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=%s", username, password, host, port, database, url.QueryEscape(timezone))

		if err := orm.RegisterDataBase("default", "mysql", dataSource); err != nil {
			logs.Error("注册默认数据库失败->", err)
			os.Exit(1)
		}

	} else if strings.EqualFold(dbadapter, "sqlite3") {

		database := beego.AppConfig.String("db_database")
		if strings.HasPrefix(database, "./") {
			database = filepath.Join(conf.WorkingDirectory, string(database[1:]))
		}
		if p, err := filepath.Abs(database); err == nil {
			database = p
		}

		dbPath := filepath.Dir(database)

		if _, err := os.Stat(dbPath); err != nil && os.IsNotExist(err) {
			_ = os.MkdirAll(dbPath, 0777)
		}

		err := orm.RegisterDataBase("default", "sqlite3", database)

		if err != nil {
			logs.Error("注册默认数据库失败->", err)
		}
	} else {
		logs.Error("不支持的数据库类型.")
		os.Exit(1)
	}
	logs.Info("数据库初始化完成.")
}

// RunCommand 注册orm命令行工具
func RegisterCommand() {

	if len(os.Args) >= 2 && os.Args[1] == "install" {
		ResolveCommand(os.Args[2:])
		Install()
	} else if len(os.Args) >= 2 && os.Args[1] == "version" {
		//CheckUpdate()
		os.Exit(0)
	}

}

//解析命令
func ResolveCommand(args []string) {
	//flagSet := flag.NewFlagSet("MinDoc command: ", flag.ExitOnError)
	//flagSet.StringVar(&conf.ConfigurationFile, "config", "", "MinDoc configuration file.")
	//flagSet.StringVar(&conf.WorkingDirectory, "dir", "", "MinDoc working directory.")
	//flagSet.StringVar(&conf.LogFile, "log", "", "MinDoc log file path.")
	//
	//if err := flagSet.Parse(args); err != nil {
	//	log.Fatal("解析命令失败 ->", err)
	//}

	if conf.WorkingDirectory == "" {
		if p, err := filepath.Abs(os.Args[0]); err == nil {
			conf.WorkingDirectory = filepath.Dir(p)
		}
	}

	if conf.ConfigurationFile == "" {
		conf.ConfigurationFile = conf.WorkingDir("conf", "app.conf")
		config := conf.WorkingDir("conf", "app.conf.example")
		if !filetil.FileExists(conf.ConfigurationFile) && filetil.FileExists(config) {
			_ = filetil.CopyFile(conf.ConfigurationFile, config)
		}
	}
	if err := gocaptcha.ReadFonts(conf.WorkingDir("static", "fonts"), ".ttf"); err != nil {
		log.Fatal("读取字体文件时出错 -> ", err)
	}

	if err := beego.LoadAppConfig("ini", conf.ConfigurationFile); err != nil {
		log.Fatal("An error occurred:", err)
	}
	if conf.LogFile == "" {
		logPath, err := filepath.Abs(beego.AppConfig.DefaultString("log_path", conf.WorkingDir("runtime", "logs")))
		if err == nil {
			conf.LogFile = logPath
		} else {
			conf.LogFile = conf.WorkingDir("runtime", "logs")
		}
	}

	conf.AutoLoadDelay = beego.AppConfig.DefaultInt("config_auto_delay", 0)
	uploads := conf.WorkingDir("uploads")

	_ = os.MkdirAll(uploads, 0666)

	beego.BConfig.WebConfig.StaticDir["/static"] = filepath.Join(conf.WorkingDirectory, "static")
	beego.BConfig.WebConfig.StaticDir["/uploads"] = uploads
	beego.BConfig.WebConfig.ViewsPath = conf.WorkingDir("views")
	beego.BConfig.WebConfig.Session.SessionCookieSameSite = http.SameSiteDefaultMode

	fonts := conf.WorkingDir("static", "fonts")

	if !filetil.FileExists(fonts) {
		log.Fatal("Font path not exist.")
	}
	if err := gocaptcha.ReadFonts(filepath.Join(conf.WorkingDirectory, "static", "fonts"), ".ttf"); err != nil {
		log.Fatal("读取字体失败 ->", err)
	}

	RegisterDataBase()
	//RegisterCache()
	RegisterModel()
	RegisterLogger(conf.LogFile)

	//ModifyPassword()

}

// RegisterLogger 注册日志
func RegisterLogger(log string) {

	logs.SetLogFuncCall(true)
	_ = logs.SetLogger("console")
	logs.EnableFuncCallDepth(true)

	if beego.AppConfig.DefaultBool("log_is_async", true) {
		logs.Async(1e3)
	}
	if log == "" {
		logPath, err := filepath.Abs(beego.AppConfig.DefaultString("log_path", conf.WorkingDir("runtime", "logs")))
		if err == nil {
			log = logPath
		} else {
			log = conf.WorkingDir("runtime", "logs")
		}
	}

	logPath := filepath.Join(log, "log.log")

	if _, err := os.Stat(log); os.IsNotExist(err) {
		_ = os.MkdirAll(log, 0755)
	}

	config := make(map[string]interface{}, 1)

	config["filename"] = logPath
	config["perm"] = "0755"
	config["rotate"] = true

	if maxLines := beego.AppConfig.DefaultInt("log_maxlines", 1000000); maxLines > 0 {
		config["maxLines"] = maxLines
	}
	if maxSize := beego.AppConfig.DefaultInt("log_maxsize", 1<<28); maxSize > 0 {
		config["maxsize"] = maxSize
	}
	if !beego.AppConfig.DefaultBool("log_daily", true) {
		config["daily"] = false
	}
	if maxDays := beego.AppConfig.DefaultInt("log_maxdays", 7); maxDays > 0 {
		config["maxdays"] = maxDays
	}
	if level := beego.AppConfig.DefaultString("log_level", "Trace"); level != "" {
		switch level {
		case "Emergency":
			config["level"] = logs.LevelEmergency
		case "Alert":
			config["level"] = logs.LevelAlert
		case "Critical":
			config["level"] = logs.LevelCritical
		case "Error":
			config["level"] = logs.LevelError
		case "Warning":
			config["level"] = logs.LevelWarning
		case "Notice":
			config["level"] = logs.LevelNotice
		case "Informational":
			config["level"] = logs.LevelInformational
		case "Debug":
			config["level"] = logs.LevelDebug
		}
	}
	b, err := json.Marshal(config)
	if err != nil {
		logs.Error("初始化文件日志时出错 ->", err)
		_ = logs.SetLogger("file", `{"filename":"`+logPath+`"}`)
	} else {
		_ = logs.SetLogger(logs.AdapterFile, string(b))
	}

	logs.SetLogFuncCall(true)
}

// RegisterModel 注册Model
func RegisterModel() {
	orm.RegisterModelWithPrefix(conf.GetDatabasePrefix(),
		new(models.Member),
		new(models.Option),
	)
	//gob.Register(models.Blog{})
	//gob.Register(models.Document{})
	//gob.Register(models.Template{})
	//migrate.RegisterMigration()
}

//系统安装.
func Install() {

	fmt.Println("Initializing...")

	err := orm.RunSyncdb("default", false, true)
	if err == nil {
		initialization()
	} else {
		panic(err.Error())
	}
	fmt.Println("Install Successfully!")
	os.Exit(0)

}

//初始化数据
func initialization() {

	err := models.NewOption().Init()
	if err != nil {
		panic(err.Error())
	}

	lang := beego.AppConfig.String("default_lang")
	err = i18n.SetMessage(lang, "conf/lang/"+lang+".ini")
	if err != nil {
		panic(fmt.Errorf("initialize locale error: %s", err))
	}

	member, err := models.NewMember().FindByFieldFirst("account", "admin")
	if errors.Is(err, orm.ErrNoRows) {

		// create admin user
		logs.Info("creating admin user")
		member.Account = "admin"
		member.Avatar = conf.URLForWithCdnImage("/static/images/headimgurl.jpg")
		member.Password = "123456"
		member.AuthMethod = "local"
		member.Role = conf.MemberSuperRole
		member.Email = "admin@iminho.me"

		if err := member.Add(); err != nil {
			panic("Member.Add => " + err.Error())
		}

		// create demo book
		//logs.Info("creating demo book")
		//book := models.NewBook()
		//
		//book.MemberId = member.MemberId
		//book.BookName = i18n.Tr(lang, "init.default_proj_name") //"MinDoc演示项目"
		//book.Status = 0
		//book.ItemId = 1
		//book.Description = i18n.Tr(lang, "init.default_proj_desc") //"这是一个MinDoc演示项目，该项目是由系统初始化时自动创建。"
		//book.CommentCount = 0
		//book.PrivatelyOwned = 0
		//book.CommentStatus = "closed"
		//book.Identify = "mindoc"
		//book.DocCount = 0
		//book.CommentCount = 0
		//book.Version = time.Now().Unix()
		//book.Cover = conf.GetDefaultCover()
		//book.Editor = "markdown"
		//book.Theme = "default"
		//
		//if err := book.Insert(lang); err != nil {
		//	panic("初始化项目失败 -> " + err.Error())
		//}
	} else if err != nil {
		panic(fmt.Errorf("occur errors when initialize: %s", err))
	}

	//if !models.NewItemsets().Exist(1) {
	//	item := models.NewItemsets()
	//	item.ItemName = i18n.Tr(lang, "init.default_proj_space") //"默认项目空间"
	//	item.MemberId = 1
	//	if err := item.Save(); err != nil {
	//		panic("初始化项目空间失败 -> " + err.Error())
	//	}
	//}
}
