# PostPath

**PostPath** is a lightweight social web application for posting thoughts, creating pages, and links for different categories. It features user authentication, a simple database backend, and a clean web interface. Built with Go and HTMX.

Visit the live site: [postpath.app](https://postpath.app)

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## Local Development

1. **Start a Go Builder Container:**
    ```bash
    cd ../Code/
    docker run --rm -it --name builder \
      -v "$PWD":/app -w /app golang:1.23 bash
    ```

2. **Install CGO Dependencies:**
    ```bash
    apt update && apt install -y gcc libc6-dev
    ```

3. **Build the Project:**
    ```bash
    go build -o main .
    ```

4. **Exit the Container:**
    ```bash
    exit
    ```

5. **Copy the Build Artifact:**
    ```bash
    docker cp builder:/app/main ./linuxBuild
    ```

6. **Start Services Locally:**
    ```bash
    docker compose up --build
    ```

---

## Production Deployment

1. **SSH Into Your Server:**
    ```bash
    ssh deploy@your.droplet.ip
    ```

2. **Install Dependencies:**
    ```bash
    sudo apt update && sudo apt install docker.io docker-compose git -y
    sudo apt install gh
    gh auth login
    ```

3. **Clone Your Repository:**
    ```bash
    cd ~
    git clone git@github.com:youruser/your-repo.git
    cd your-repo
    ```

4. **SSL Setup:**
    - Add Cloudflare Origin Server certificates to the `certs` directory.

5. **mTLS Setup:**
    - Get origin cert from [Cloudflare Zone-Level Authenticated Origin Pull](https://developers.cloudflare.com/ssl/origin-configuration/authenticated-origin-pull/set-up/zone-level/).

6. **Deploy with Docker Compose:**
    ```bash
    docker-compose down
    docker-compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d --remove-orphans
    ```

---

*© 2025 PostPath. For questions or support, contact the maintainer.*