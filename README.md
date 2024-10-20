# LoThone Panel
This is the v2ray server manager of vmess config namely LoThone panel.

# Development
## How to run

- type the following command to start the hot reloading.
```bash
air
```
- The auto reloading will work in `localhost:8090`
- The server will run on port `:8888` and it's being mapped to `:8090` for convenience.

## Installing v2ray-server-manager
### Prerequisites
- A working V2Ray server of version 4.31.0. which is hosted in this [link](https://github.com/v2fly/v2ray-core/releases/tag/v4.31.0])
- Root or sudo access to the server.

1) Create the User
    - Create the user:
    ```bash
    sudo useradd -r -s /usr/sbin/nologin v2rayadmin
    ```
    This creates a system user named v2rayadmin without a home directory and prevents it from logging in interactively.


    - Set a password (optional): If you want to set a password for administrative purposes (not for login):
    ```bash
    sudo passwd v2rayadmin
    ```
2) Install and Configure Sudo
    - Install sudo (if not already installed):
    ```bash
    sudo apt-get install sudo
    ```

    - Edit the sudoers file: Open the sudoers file with the visudo command to prevent syntax errors:
    ```bash
    sudo visudo
    ```
    Add the following lines to allow v2rayadmin to run specific commands without a password:
    ```text
    v2rayadmin ALL=(ALL) NOPASSWD: /bin/systemctl restart v2ray, /path/to/v2ray -test -config /path/to/config/file
    ```
    Replace /path/to/v2ray and /path/to/config/file with the actual paths to your V2Ray executable and configuration file.
    Save and exit: Save the changes in visudo. This allows v2rayadmin to restart the V2Ray service and test the configuration without being prompted for a password.


3) Create the Systemd Unit File Or Use the one that we provided
    - If you haven't created a systemd unit file for V2Ray server manager, follow these steps:
    Create the unit file:
    ```bash
    sudo nano /etc/systemd/system/v2ray.service
    ```
    - Add the following content and configure:
    ```text
    [Unit]
    Description=v2ray server-manager
    After=network.target

    [Service]
    Type=simple
    ExecStart=/path/to/project-root/server-manager-bin/server-manager \ 
            -admin="admin@lothone.shop" \
            -adminpw="774c8f08-f500-49c2-a00b-68de23aa0070" \
            -configfile="" \
            -userfile="" \
            -hostip="" \
            -hostname="" \
            -v2rayport="443" \
            -webcert="" \
            -webkey="" \
            -webport="8080"
    Restart=on-failure
    RestartSec=5
    User=v2rayadmin
    Group=v2rayadmin
    WorkingDirectory=/path/to/project-root
    ExecStartPre=/usr/bin/some-command  # Optional: Run config v2ray server config checking.
    StandardOutput=journal
    StandardError=journal
    LimitNOFILE=4096
    TimeoutStartSec=30
    TimeoutStopSec=30

    [Install]
    WantedBy=multi-user.target
    ```
    or Use the one in the systemd folder.
    - Enable and start the service:
    ```bash
    sudo systemctl enable v2ray-server-manager
    sudo systemctl start v2ray-server-manager
    ```
4) Testing the Configuration
    - Log in as v2rayadmin (if necessary, you can switch users):
    ```bash
    sudo -u v2rayadmin -s
    ```
    - Test the V2Ray configuration: Use the command defined in the sudoers file:
    ```bash
    sudo /path/to/v2ray -test -config /path/to/config/file
    ```
    - This will test the V2Ray configuration without needing a password.
    - Restart the V2Ray service: You can restart the V2Ray service using:
    ```bash
    sudo systemctl restart v2ray-server-manager
    ```

5) Automation
    - V2ray server manager uses certificates to encrypt the connection between the clients and the server. You can create the certificates signed by [letsencrypt](https://letsencrypt.org) using [certbot](https://certbot.eff.org).
    ```bash
    sudo apt install certbot
    ```

    1) Using certbot only

        ```bash
        sudo certbot certonly --standalone --preferred-challenges http --agree-tos --key-type rsa --email <your_email_address> -d <your_full_domain_name>
        ```

    2) Using nginx plugins

        ```bash
        sudo vim /etc/nginx/conf.d/<your_full_domain_name>.conf
        ```
        paste the following lines into the file.
        ```bash
        server {
              listen 80;
              server_name <your_full_domain_name>;

              root /var/www/html/;

              location ~ /.well-known/acme-challenge {
                 allow all;
              }
        }
        ```
        create the web root directory.
        ```bash
        sudo mkdir -p /var/www/html
        ```
        www-data(nginx user) as the owner of the web root.
        ```bash
        sudo chown www-data:www-data /var/www/html -R
        ```
        reload nginx for the changes to take effect.
        ```bash
        sudo systemctl reload nginx
        ```
        create the certificate
        ```bash
        sudo certbot certonly --webroot --agree-tos --key-type rsa --email <your_email_address> -d <your_full_domain_name> -w /var/www/html
        ```

    Since letsencrypt certificates do expire you have to renew the certificates. You can do that by creating a `cron` job.
    - For starter, run the following command to se the cron server is running
    ```bash
    sudo systemctl status cron
    ```
    - Open the crontab using the following command.
    ```bash
    sudo crontab -e
    ```
    - Add the Cron job, for example to renew the server-manager every day, add the following line.
    ```txt
    @daily certbot renew --quiet; systemctl restart server-manager
    ```
    - Verify the cronjob you have just added.
    ```bash
    sudo crontab -l
    ```
