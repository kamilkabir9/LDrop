# LDrop
[![IMAGE LDrop demo](http://img.youtube.com/vi/Njlo46ngp5E/0.jpg)](http://www.youtube.com/watch?v=Njlo46ngp5E)
LDrop helps you to serve directory from your computer to devices in the local network.Files can be viewed ,uploaded and downloaded through web frontEnd.
## Getting Started
LDrop have the following options.
```
  -ignoreHiddenFiles
    	Pass True to ignore hidden Folders
  -ignoreHiddenFolders
    	Pass True to ignore hidden Files
  -ignorePreffix value
    	Pass file PREFFIX to exclude Example:"PIC-,MOV-"
  -ignoreSuffix value
    	Pass file SUFFIX to exclude Example:".png,.mp4"
  -onlySuffix value
    	Pass file SUFFIX to only to include
  -secret string
    	Pass secret code. (default "007Jb")
  -v	Pass true to print verbose info
```
### Installing
 - Download a respective binary from [here](https://github.com/kamilkabir9/LDrop/releases/latest) .
 - (Optional) Add LDrop to your path .
 - Run it.
 - Scan the QRcode on you mobile ;to view or upload files.
Example: ```LDrop -v -onlySuffix=.mp4 ./Downloads```

## License
This project is licensed under the GNU GPLv3 - see the [LICENSE.md](LICENSE.md) file for details
