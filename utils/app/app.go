package app

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gookit/goutil/fsutil"
	"github.com/redis/go-redis/v9"
	"github.com/sevlyar/go-daemon"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"xorm.io/xorm"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type Mode string

const (
	ModeProd  Mode = "prod"
	ModeDev   Mode = "dev"
	ModeDebug Mode = "debug"
	ModeDemo  Mode = "demo"
)

var (
	Version        = "v0.0.0"
	Goversion      = ""
	Commit         = ""
	Build          = ""
	RunMode   Mode = ModeProd
)

func ShowVersion() {
	fmt.Println("version:", Version)
	fmt.Println("go:", Goversion)
	fmt.Println("commit:", Commit)
	fmt.Println("build:", Build)
}

func ConfigInit(fp string) {
	viper.SetConfigFile(fp)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	//
	time.LoadLocation(viper.GetString("main.time_zone"))

	RunMode = Mode(viper.GetString("main.mode"))
}

func RedisInit() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     viper.GetString("redis.host"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),

		DialTimeout:           10 * time.Second,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		ContextTimeoutEnabled: true,

		MaxRetries: -1,

		PoolTimeout:     30 * time.Second,
		ConnMaxIdleTime: time.Minute,

		PoolSize:     15,
		MinIdleConns: 10,

		ConnMaxLifetime: 0,
	})
}

func LogsInit(fn string, is_daemon bool) {
	level, _ := logrus.ParseLevel(viper.GetString("main.log_level"))
	logrus.SetLevel(level)

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceQuote:      true,                  //键值对加引号
		TimestampFormat: "2006-01-02 15:04:05", //时间格式
		FullTimestamp:   true,
	})

	output := []io.Writer{}
	if !is_daemon {
		output = append(output, os.Stdout)
	}
	if viper.GetString("main.log_path") != "" {
		save_path := viper.GetString("main.log_path")
		err := fsutil.Mkdir(save_path, 0755)
		if err != nil {
			logrus.Fatal(err)
		}

		file := fmt.Sprintf("%s/%s_%d.log", save_path, fn, time.Now().Unix())
		f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			logrus.Fatal(err)
		}
		output = append(output, f)
	}

	mw := io.MultiWriter(output...)
	logrus.SetOutput(mw)
}

func DatabaseInit() *xorm.Engine {
	dsn := viper.GetString("database.dsn")
	driver := viper.GetString("database.driver")
	conn, err := xorm.NewEngine(driver, dsn)
	if err != nil {
		logrus.Panic(err)
	}

	// tbMapper := core.NewPrefixMapper(core.SnakeMapper{}, "ex_")
	// conn.SetTableMapper(tbMapper)
	if viper.GetBool("database.show_sql") {
		conn.ShowSQL(true)
	}

	conn.DatabaseTZ = time.Local
	conn.TZLocation = time.Local
	return conn
}

func Deamon(pid string, logfile string) (*daemon.Context, *os.Process, error) {
	cntxt := &daemon.Context{
		PidFileName: pid,
		PidFilePerm: 0644,
		LogFileName: logfile,
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		// Args:        ,
	}

	child, err := cntxt.Reborn()
	return cntxt, child, err
}
