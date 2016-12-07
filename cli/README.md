# Event Driven Adaptor

事件驅動轉接器

### Eample 

#### shell script

```sh
./eventd -event [channel name1] -event [channel name2] ./script.sh 1>>/tmp/info.log 2>>/tmp/err.log 
```

#### php

```sh
./eventd -event [channel name] php ./script.php
```
or
```sh
./eventd -event [channel name] ./script-cli.php
```


