# Installation instructions

Due to extremely tight security restrictions on recent versions of macOS, and the unconventional data storage model of this app (everything within the directory where the app resides instead of scattered around ~/Library, ~/Documents, etc.), you need to jump through hoops to make the app work.

Assuming you downloaded EggLedger-mac.zip to your Downloads folder and unpacked it there, you should be able to find the app at EggLedger/EggLedger.app in Downloads. You need to open Terminal.app [1], enter the following command, and press return:

    xattr -c ~/Downloads/EggLedger/EggLedger.app

This command removes the "com.apple.quarantine" attribute from the app, which otherwise causes the app to be run in a readonly jail without the ability to write data to the appropriate folder.

If you downloaded and unpacked the app to a different location, you will need to change the path in the command above.

When you launch the app for the first time, instead of double clicking, you may need to right click (control click) and choose "Open". Otherwise you may be asked to move the app to trash because the app is unsigned -- yes, it's unsigned because I'm not in the $100/yr Apple Developer Program.

Finally, it has been reported that you may not be able to run the app at all on some older versions of macOS, asking you to upgrade your operating system. As a workaround, you can right click -> "Show Package Contents", then run the Contents/MacOS/EggLedger executable inside directly. You may need to similarly right click -> "Open" when running it for the first time.

[1] You can launch Terminal.app from Spotlight, Launchpad, /Applications/Utilities. See https://support.apple.com/guide/terminal/open-or-quit-terminal-apd5265185d-f365-44cb-8b09-71a064a42125/mac
