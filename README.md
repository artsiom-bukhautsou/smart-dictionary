Build an image:
```bash
docker build -t smart-dict .
```

Run app locally via docker
```bash
docker run --env-file .env.tmp -p 8080:8080 smart-dict
```