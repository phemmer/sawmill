The writer handler package is responsible for writing events to standard IO streams (`io.Writer`).

There are a few convenience functions for constructing handlers which write to STDOUT/STDERR and appending to normal files.

Examples:

### STDOUT/STDERR

    logger := sawmill.NewLogger()
    logger.AddHandler("stdStreams", writer.NewStandardStreamsHandler())
    
    # will go to STDERR
    logger.Warning("foo", sawmill.Fields{"bar": "baz"})
    
    # will go to STDOUT
    logger.Info("foo", sawmill.Fiels{"pop": "tart"})
    
    logger.Stop()

### File appender

    logger := sawmill.NewLogger()
    h, err := writer.Append("/var/log/foo", 0600, "")
    if err != nil {
    	sawmill.Panic("error opening log file", sawmill.Fields{"error": err, "path": "/var/log/foo"})
    }
    logger.AddHandler("logfile", h)
    
    logger.Info("FOO!", sawmill.Fields{"bar": "baz"})
