# Usage

1. Copy collected data (zip file) to `data` folder
2. Extract tarball in `data` folder
    ```shell script 
    unzip -x [timestamp]-data.zip
    ```
    _Example_
    ```
    .../data$ unzip -x [timestamp]-data.zip
    ```
3. Change metrics snapshot owner 
    ```shell script
    sudo chown -R 65534:65534 ./metrics/snapshot/
    ```
    _Example_
    ```
    .../data$ sudo chown -R 65534:65534 ./metrics/snapshot/
    ``` 
4. Start docker compose
    ```shell script
    .../analysis$ docker-compose up
    ```
    _Example_
    ```shell script
    docker-compose up
    ```
5. Open Graphana page http://localhost:3000