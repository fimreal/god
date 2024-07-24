# go build
#FROM golang:1.21 AS builder
#RUN git clone https://github.com/fimreal/god.git /god &&\
#    cd /god &&\
#    CGO_ENABLED=0 GOOS=linux go build -ldflags '-s -w' -o god .
# or CGO_ENABLED=0 GOOS=linux go build -o god .

# openresty & php
FROM openresty/openresty:alpine

# Motify timezone as needed
# ENV TZ="Asia/Shanghai"

# Apk mirrors proxy in China
# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# default: php83
RUN apk add --no-cache php-fpm php-pdo_mysql php-tokenizer \
    php-xml php-pdo php-phar php-openssl php-pdo_pgsql\
    php-json php-pdo_mysql php-mysqli php-session \
    php-pdo_sqlite php-sqlite3 php-exif php-intl \
    php-gd php-iconv php-gmp php-zip \
    php-curl php-opcache php-ctype php-apcu \
    php-intl php-bcmath php-dom php-mbstring php-xmlreader \
    php-pecl-memcache php-pecl-redis php-ftp \
    curl openssl tzdata &&\
    sed -i 's/expose_php = On/expose_php = Off/g;s/post_max_size = 8M/post_max_size = 0/g;s/upload_max_filesize = 2M/upload_max_filesize = 5G/g' /etc/php83/php.ini

ADD ./nginx.conf /etc/nginx/conf.d/nginx.conf

# Copy god bin command
# COPY --from=builder /god/god /god
COPY --from=epurs/god:latest /god /god
CMD ["/god", "-c", "nginx:openresty -g 'daemon off;'", "-c", "/usr/sbin/php-fpm83 --nodaemonize"]

# health check, optional
HEALTHCHECK --interval=5s --timeout=2s --retries=3 CMD curl -f http://localhost:7788/health || exit 1

EXPOSE 80 7788
VOLUME ["/wwwroot"]