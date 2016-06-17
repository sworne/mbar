package main
import (
    "os"
    "os/exec"
     "bufio"
    "flag"
    "log"
    "fmt"
    "strconv"
    "strings"
    "github.com/fhs/gompd/mpd"
    "time"
    "regexp"
)

type Bar struct {
    fg string
    rst string
    bg []string
    flag map[string]string
    icn map[string]string
    cmd map[string]string
    clock chan string
    cal chan string
    desktop chan int
    batt_level chan int
    mpd chan string
    bar []string
      }

func NewBar(fg string, bg []string, icn, cmd map[string]string, flag map[string]string) *Bar {
    b := new(Bar)
    b.bg = bg
    b.cmd = cmd
    for i := range b.bg {
      b.bg[i] = GenColorTag(b.bg[i], false)
    }
    for i := range b.cmd {
        b.cmd[i] = GenCmdTag(b.cmd[i])
    }
    b.cmd["rst"] = "%{A}"
    b.fg = GenColorTag(fg, true)
    b.rst = GenColorTag("#00000000", false)
    b.icn = icn
    b.flag = flag
    b.clock = make(chan string, 1)
    b.cal = make(chan string, 1)
    b.desktop = make(chan int, 1)
    b.batt_level = make(chan int, 1)
    b.mpd = make(chan string, 1)

    x := "xprop -root _NET_CURRENT_DESKTOP"
    mpd_con := flag["mpd_host"] + ":" + flag["mpd_port"]
    //center := "%{c}"
    second := time.Second
    minute := time.Minute

    go Clock(second * 10, b.clock)
    go Month(minute * 5, b.cal)
    go Desktop(second, x, b.desktop)
    go MpdStatus(second, mpd_con, b.mpd)
    go BattLevel(second * 10, flag["batt_filename"], b.batt_level)
    //go BattStatus(second * 20, flag["batt_filename"], b.batt_charge)

    return b
}

func (bar Bar) Print() {
  de, batt, clock, month, m, mpd_state := "", "", "", "", "", ""
  light := fmt.Sprintf("%s %s %s %s %s %s %s", bar.cmd["light_down"], bar.icn["down"], bar.cmd["rst"], bar.icn["leaf"], bar.cmd["light_up"], bar.icn["up"], bar.cmd["rst"])
  vol := fmt.Sprintf("%s %s %s %s %s %s %s", bar.cmd["vol_down"] , bar.icn["down"] , bar.cmd["rst"] , bar.icn["vol"] , bar.cmd["vol_up"] , bar.icn["up"] , bar.cmd["rst"])
  lock := fmt.Sprintf(" %s%s%s ", bar.cmd["lock"] , bar.icn["lock-alt"] , bar.cmd["rst"])
  mpd_con := bar.flag["mpd_host"] + ":" + bar.flag["mpd_port"]


for {
  select {
    case mpd_state = <- bar.mpd:
      switch mpd_state {
        case "play":
          p := MpdPlaying(mpd_con)
          p = Truncate(p, 60)
          m = fmt.Sprintf(" %s %s %s %s %s %s %s ", bar.cmd["mpd_pause"], bar.icn["peace"], bar.cmd["rst"], p , bar.cmd["mpd_next"], bar.icn["right"], bar.cmd["rst"])
        case "pause":
          m = fmt.Sprintf(" %s %s %s ", bar.cmd["mpd_play"], bar.icn["peace"], bar.cmd["rst"])
        case "stop":
          m = fmt.Sprintf(" %s %s %s ", bar.cmd["mpd_start"], bar.icn["stop"], bar.cmd["rst"])
        case "off":
          m = fmt.Sprintf(" %s ", bar.icn["fist"])
        }
    case d := <- bar.desktop:
      de = fmt.Sprintf(" %s%s ", bar.icn["desktop"], strconv.Itoa(d))
    case b := <- bar.batt_level:
        if BattStatus(bar.flag["batt_filename"]) {
          batt = fmt.Sprintf(" %s %s ", bar.icn["circle"], strconv.Itoa(b))
        } else {
          switch {
          case b < 20:
            batt = fmt.Sprintf(" %s %s ", bar.icn["star_1"], strconv.Itoa(b))
          case b < 70:
            batt = fmt.Sprintf(" %s %s ", bar.icn["star_2"], strconv.Itoa(b))
          case b <= 100:
            batt = fmt.Sprintf(" %s %s ", bar.icn["star_3"], strconv.Itoa(b))
        }
      }
    case c := <- bar.clock:
      clock = fmt.Sprintf(" %s %s ", bar.icn["clock"], c)
    case m := <- bar.cal:
      month = fmt.Sprintf(" %s %s ", bar.icn["month"], m)
  }

  line := ""
  l := []string{de, batt, clock, month, light, vol, lock , m}

  line = line + bar.fg
  for i := range l {

        line = line + fmt.Sprintf("%s %s ", bar.bg[i] , l[i])
      }
  line = line + bar.rst
  fmt.Println(line)
  }
}

func main () {
    // FLAGS
    batt_filename := flag.String("batt-override", "/sys/class/power_supply/BAT0", "Overide default batt directory")
    mpd_host := flag.String( "host", "localhost", "mpd host address")
    mpd_port := flag.String( "port", "6600", "mpd host port")
    hex_bg := flag.String( "bg", "#1f1f1f", "hex value for background")
    hex_fg := flag.String( "fg", "#c0b18b", "hex value for text")
    flag.Parse()
    flags := map[string]string{
      "batt_filename" : *batt_filename,
      "mpd_host" : *mpd_host,
      "mpd_port" : *mpd_port,
    }

    // ICONS--------------------------
    icn := map[string]string{
    "desktop" : "\uf120",
    "month" : "\uf186",
    "clock" : "\uf017",
    "lock" : "\uf023",
    "lock-alt" : "\uf070",
    "vol" : "\uf0a1",
    "up" : "\uf102",
    "down" : "\uf103",
    "leaf" : "\uf06c",
    "star_1" : "\uf006",
    "star_2" : "\uf123",
    "star_3" : "\uf005",
    "circle" : "\uf10c",
    "peace" : "\uf25b",
    "stop" : "\uf256",
    "right" : "\uf0a4",
    "fist" : "\uf088",
}
    // COMMANDS------------------------
    cmd := map[string]string{
    "lock": "$HOME/bin/lock.sh",
    "mpd_next":   "mpc next",
    "mpd_pause": "mpc pause",
    "mpd_play": "mpc play",
    "mpd_start": "mpc load \"Discover Weekly (by spotifydiscover)\" && mpc play && mpc shuffle",
    "vol_up": "pactl set-sink-volume 1 +10%",
    "vol_down":   "pactl set-sink-volume 1 -10%",
    "light_up": "xbacklight +10",
    "light_down":  "xbacklight -10",
    }
    // BG--------------------------
    bg := GenHex(*hex_bg)

    // FG------------------------------
    fg := *hex_fg

    // BAR-----------------------------
    bar := NewBar(fg, bg, icn, cmd, flags)
    bar.Print()
}

func GenColorTag(s string, fg bool) (tag string) {
    if fg == true {
        tag = "%{F" + s + "}"
    } else {
        tag = "%{B" + s + "}"
    }
    return
}

func GenHex(s string) (l []string) {
      //Remove # from s
      l = []string{}
      s = fmt.Sprintf(strings.Replace(s, "#", "", 2))

      // Convert to hash value
      n, err := strconv.ParseUint(s, 16, 32)
      if err != nil {
          log.Fatalln(err)
      }

      // Generate
      i := 0
      for i < 15 {
          l = append(l,"#" + strconv.FormatUint(n, 16))
          n = n + 657930
          i = i + 1
      }
      return
}

func GenCmdTag(s string) (tag string) {
    tag = "%{A:" + s + ":}"
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

func Desktop(duration time.Duration, cmd string, ch chan int) () {
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
    ch <- desktop_no + 1
    time.Sleep(duration)
  }
}

func MpdStatus(duration time.Duration, con string, ch chan string) () {
  conn, err := mpd.Dial("tcp", con)
  if err != nil {
    log.Fatalln(err)
  }
  defer conn.Close()
    for {
        status, err := conn.Status()
        if err != nil {
          ch <- "off"
          } else {
          ch <- status["state"]
        }
        time.Sleep(duration)
    }
}

func MpdPlaying(con string) (p string) {
  conn, err := mpd.Dial("tcp", con)
  if err != nil {
    log.Fatalln(err)
  }
  defer conn.Close()
  status, err := conn.Status()
  if err != nil {
      log.Fatalln(err)
      }
  song, err := conn.CurrentSong()
  if err != nil {
    log.Fatalln(err)
    }
    if status["state"] == "play" {
      p = fmt.Sprintf("%s - %s", song["Artist"], song["Title"])
      } else {
        p = "nothing playing"
      }
      return
    }



func BattLevel( duration time.Duration, f string, ch chan int) () {
  f = f + "/capacity"
  for {
    s := Cat(f)
    l, err := strconv.Atoi(s)
    if err != nil {
        log.Fatal(err)
    }
    ch <- l
    time.Sleep(duration)
  }
}

func BattStatus( f string) ( c bool) {
        f = f + "/status"
        s := Cat(f)
        if s == "Charging" {
            return true
        } else {
            return false
        }
}

func Truncate( s string, l int ) (o string) {
    i := 0
    for index, _ := range s {
         i++
         if i > l {
              return s[:index]
         }
    }
    return s
}

func Cat( f string) (c string) {
    file, err := os.Open(f)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    s := bufio.NewScanner(file)
    for s.Scan() {
        c = s.Text()
    }
    if err := s.Err(); err != nil {
        log.Fatal(err)
    }
    return
}
