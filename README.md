# ylog
a golang implementation of logging module.

It provides functions Debug, Trace, Warn, Error,

plus formatting variants such as Debugf. Logs will split

into several files according to log time and file size.

Log output is buffered and written periodically using Flush. Programs

should call Flush before exiting to guarantee all log output is written.

Basic example:

    func main() {
        flag.Parse()            // parse flags
        ylog.Init()             // initialize ylog system
        defer ylog.Flush()      // ensure flush before exit
        ylog.Debug("debug")
        ylog.Trace("trace")
        ylog.Warnf("Process failed: %s", err)
    }



