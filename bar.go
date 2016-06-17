package main
import (
    "os"
    "os/exec"
     "bufio"
    "flag"
    "log"
    "fmt"
    "strconv"
    "github.com/fhs/gompd/mpd"
    "time"
    "regexp"
)

type Bar struct {
  month string
  clock string
  batt_icon string
  mpd_icon string
  mpd_playing string
  batt_level int
  desktop_no int
  icon  map[string]string
      }


func main () {
    // FLAGS
    batt_filename := flag.String("batt-override",
       "/sys/class/power_supply/BAT0/capacity", "Overide default batt directory")
    mpd_host := flag.String( "host", "localhost", "mpd host address")
    mpd_port := flag.String( "port", "6600", "mpd host port")
    flag.Parse()

    // CHANELS--------------------
    clock_c := make(chan string)
    month_c := make(chan string)
    desktop_no_c := make(chan int)
    batt_icon_c := make(chan string)
    batt_level_c := make(chan int)
    mpd_icon_c := make(chan string, 300)
    mpd_playing_c := make(chan string, 300)
    mpd_cmd_c := make(chan string, 300)

    // VARS--------------------------
    bg := []string{"#1f1f1f", "#292929", "#363636", "#3D3D3D", "#474747",
                    "#525252","#5C5C5C", "#666666", "#707070", "#7A7A7A"}
    x := "xprop -root _NET_CURRENT_DESKTOP"
    mpd_con := *mpd_host + ":" + *mpd_port
    fg := GenColorTag("#c0b18b", true)
    rst := GenColorTag("#00000000", false)
    //center := "%{c}"
    second := time.Second
    minute := time.Minute

    for i := range bg {
      bg[i] = GenColorTag(bg[i], false)
    }

    conn, err := mpd.Dial("tcp", mpd_con)
    if err != nil {
      log.Fatalln(err)
    }
    defer conn.Close()

    // GOROUTINES-----------------------
    go Clock(second * 10, clock_c)
    go Month(minute * 5, month_c)
    go Desktop(second, x, desktop_no_c)
    go MpdStatus(second, *conn, mpd_playing_c, mpd_icon_c, mpd_cmd_c)
    go Batt(second * 30, *batt_filename, batt_level_c, batt_icon_c )

    // ICONS--------------------------
    icons := make(map[string]string)
    icons["desktop"] = "\uf120"
    icons["month"] = "\uf186"
    icons["clock"] = "\uf017"
    icons["chrome"] = "\uf14e"
    icons["lock"] = "\uf023"
    icons["lock-alt"] = "\uf070"
    icons["circ"] = "\uf10c"
    icons["atom"] = "\uf121"
    icons["batt_icon"] = <-batt_icon_c
    icons["mpd_icon"] = <-mpd_icon_c
    icons["vol"] = "\uf0a1"
    icons["up"] = "\uf102"
    icons["down"] = "\uf103"
    icons["leaf"] = "\uf06c"



    // COMMANDS------------------------
    cmd := make(map[string]string)
    cmd["lock"] = "%{A:$HOME/bin/lock.sh:}"
    cmd["chrome"] = "%{A:google-chrome-stable:}"
    cmd["atom"] = "%{A:atom:}"
    cmd["mpd"] =  <-mpd_cmd_c
    cmd["rst"] = "%{A}"
    cmd["vol_up"] = "%{A:pactl set-sink-volume 1 +10%:}"
    cmd["vol_down"] = "%{A:pactl set-sink-volume 1 -10%:}"
    cmd["light_up"] = "%{A:xbacklight +10:}"
    cmd["light_down"] = "%{A:xbacklight -10:}"

    // CHANNELS----------------------
    str := make(map[string]string)
    ints := make(map[string]int)

    str["month"] = <-month_c
    str["clock"] = <-clock_c
    str["mpd_playing"] = <-mpd_playing_c
    ints["batt_level"] = <-batt_level_c
    ints["desktop_no"] = <-desktop_no_c

    //instance := &T{Name: "foo", Versions: map[byte]version{}}

    for {
      select {
      case ints["desktop_no"] = <-desktop_no_c:
          Print(bg, icons, cmd, str, ints, fg, rst)
      case ints["batt_level"] = <-batt_level_c:
          icons["batt"] = <-batt_icon_c
          Print(bg, icons, cmd, str, ints, fg, rst)
      case str["clock"] = <-clock_c:
          Print(bg, icons, cmd, str, ints, fg, rst)
      case str["month"] = <-month_c:
          Print(bg, icons, cmd, str, ints, fg, rst)
      case str["mpd_playing"] = <-mpd_playing_c:
          cmd["mpd"] = <-mpd_cmd_c
          icons["mpd_icon"] = <-mpd_icon_c
          Print(bg, icons, cmd, str, ints, fg, rst)
     }
  }
}

func Print(bg []string, icons, cmd, str map[string]string, ints map[string]int, fg, rst string) {
  fmt.Println (
          fg,
          bg[0], icons["desktop"], ints["desktop_no"],
          bg[1], icons["batt_icon"], ints["batt_level"],
          bg[2], icons["clock"], str["clock"],
          bg[3], icons["month"], str["month"],
          //bg[4], cmd["chrome"], icons["chrome"], cmd["rst"],
          //bg[5], cmd["atom"], icons["atom"], cmd["rst"],
          bg[5], cmd["light_down"], icons["down"], cmd["rst"],
          bg[5], icons["leaf"],
          bg[5], cmd["light_up"], icons["up"], cmd["rst"],
          bg[6], cmd["vol_down"], icons["down"], cmd["rst"],
          bg[6], icons["vol"],
          bg[6], cmd["vol_up"], icons["up"], cmd["rst"],
          bg[7], cmd["lock"], icons["lock-alt"], cmd["rst"],
          bg[8], cmd["mpd"], icons["mpd_icon"], cmd["rst"], str["mpd_playing"],
          rst)
}


func GenColorTag(color string, fg bool) (color_code string) {
    if fg == true {
        color_code = "%{F" + color + "}"
    } else {
        color_code = "%{B" + color + "}"
    }
    return
}

func Clock(duration time.Duration, ch chan string) () {
  for {
  t := time.Now()
  ch <- fmt.Sprint(t.Format(time.Kitchen))
  time.Sleep(duration)
  }
}

func Month(duration time.Duration, ch chan string) () {
  for {
		_, month, day := time.Now().Date()
    ch <- fmt.Sprint(month, day)
    time.Sleep(duration)
    }
  }

func Desktop(duration time.Duration, cmd string, desktop_no_c chan int) () {
  for {
    out, err := exec.Command("sh", "-c", cmd).Output()
    if err != nil {
        log.Fatalln(err)
    }
    out_str := fmt.Sprintf("%s", out)
    re := regexp.MustCompile("[0-9]+")
    desktop_str := re.FindAllString( out_str, -1)[0]
    desktop_no, err := strconv.Atoi(desktop_str)
        if err != nil {
        log.Fatal(err)
    }
    desktop_no_c <- desktop_no + 1
    time.Sleep(duration)
  }
}

func MpdStatus(duration time.Duration, conn mpd.Client, line_c, icon_c, cmd_c chan string) () {
    icon_play := "\uf25b"
    icon_stop := "\uf256"
    icon_next := "\uf0a4"
    icon_off := "\uf088"

    line := ""
    icon := "\uf256"
    cmd := "%{A:mpc load \"Discover Weekly (by spotifydiscover)\" && mpc play && mpc shuffle:}"

    for {
        status, err := conn.Status()
        if err != nil {
            icon = icon_off
            line = ""
            cmd = "%{A::}"
        } else {
        song, err := conn.CurrentSong()
        if err != nil {
            log.Fatalln(err)
        }

        switch status["state"] {
          case "play":
            line = fmt.Sprintf("%s - %s", song["Artist"], song["Title"])
            icon = icon_next
            cmd = "%{A:mpc next:}"
          case "pause":
            cmd = "%{A:mpc play:}"
            line = ""
            icon = icon_play
          case "stop":
            cmd = "%{A:mpc load \"Discover Weekly (by spotifydiscover)\" && mpc play && mpc shuffle:}"
            line = ""
            icon = icon_stop
            }
          }

        cmd_c <- cmd
        line_c <- line
        icon_c <- icon
        time.Sleep(duration)
    }
}


func Batt( duration time.Duration, filename string, level_c chan int, icon_c chan string ) () {
  for {
    batt_1 := "\uf006"
    batt_2 := "\uf123"
    batt_3 := "\uf005"
    icon := batt_3

    level_str := Cat(filename)
    level, err := strconv.Atoi(level_str)
    if err != nil {
        log.Fatal(err)
    }

    switch {
    case level < 20:
            icon = batt_1
    case level < 70:
            icon = batt_2
    case level < 100:
            icon = batt_3
    }
    icon_c <- icon
    level_c <- level
    time.Sleep(duration)
  }
}

func Cat( filename string) (file_content string) {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        file_content = scanner.Text()
    }
    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
    return
}
