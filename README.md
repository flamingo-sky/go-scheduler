# go-scheduler
a go scheduler


依赖于 <br>
cron:      https://github.com/robfig/cron <br>
drumstick: https://github.com/openex27/drumstick <br>
提供策略开始时间，策略间隔时间<br>
欢迎提issue<br>
示例:

    func main() {
        cron := New()
        s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
        cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
            time.Now().Format("2006-01-02 15:04:05")) }, "test1")
        cron.Start()
        defer cron.Stop()
    
        // Give cron 20 seconds to run our job (which is always activated).
        select {
        case <-time.After(20*ONE_SECOND):
        }
    }
