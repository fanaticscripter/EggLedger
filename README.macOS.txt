Installation instructions
=========================

Before you run the app for the first time, you need to do the following to mark the app as safe, otherwise the app won't function correctly:

* [underline green]#*Right click on the `preflight` script, choose "Open" and let the script run.*# `EggLedger.app` must not be moved to elsewhere at this point.

You should be able to launch the app normally now.

You can also move the entire folder containing `EggLedger.app` to anywhere you like. Do NOT move the app separately to a shared folder like `/Applications`; the app writes data to the folder where `EggLedger.app` resides, you don't want to pollute a shared folder.

[WARNING]
If you encounter the [red]#"... cannot be opened because the developer cannot be verified"# error when launching either `preflight` or `EggLedger.app`, [green underline]#*instead of double clicking, right click on the script or app and choose "Open".*# The error occurs because the programs are not signed with a recognized developer certificate (only available to developers enrolled in Apple's paid Developer Program).
