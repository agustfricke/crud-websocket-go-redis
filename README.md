# WEBSOCKET CRUD WITH REDIS AND GO

-   Use a key-value format similar to JSON, where we separate the values with ||, e.g.: id: 23 || title: hello!! || action: normal.

## How to use

-   Run a redis db(you can do so with Docker)

```bash
sudo docker run --name mi-redis -d -p 6379:6379 redis
```

-   Clone the repo and run

```bash
git clone https://github.com/agustfricke/crud-websocket-go-redis.git
cd crud-websocket-go-redis
go run main.go
```

# How to deploy a WSS (WebSocket Secure) with Nginx?

-   Assuming you already have Nginx installed and have enabled ports 80, 443, and 8080 on your server, let's proceed directly to the configuration.

-   The first step is to create a certificate with Let's Encrypt.

```bash
sudo apt install certbot python3-certbot-nginx
certbot --nginx
```

-   Accept the terms and conditions, provide your email, and specify the domain for which you want to create a certificate.

-   Now, in a terminal, run the Go server. The server will listen on port 8080.

```bash
go run main.go
```

-   In the path **/etc/nginx/sites-available**, create a file called **websocket-fiber.conf**.

```bash
server {
    listen 443 ssl;
    server_name www.your-domain.com your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem; # managed by Certbot

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

-   Make sure to replace "your-domain.com" with your actual domain.

-   Now, let's place this configuration in sites-enabled and remove the Nginx default configuration.

```bash
sudo rm /etc/nginx/sites-available/default /etc/nginx/sites-enabled/default
sudo ln -s /etc/nginx/sites-available/websocket-fiber.conf /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

Now, if you visit your domain, you should have a chat using WSS (WebSocket Secure) and HTTPS.

# Enjoy and give it a star
