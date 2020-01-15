# Docker

```shell script
sudo usermod -a -G docker $USER
```

version: '3'
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    user: root
    ports:
      - "9090:9090"
    volumes:
      - /home/mars/sources/supportCenter/agent/data/20200115T150048/10.0.0.239:/etc/prometheus
      - /home/mars/sources/supportCenter/agent/data/20200115T150048/10.0.0.239/snapshot:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'