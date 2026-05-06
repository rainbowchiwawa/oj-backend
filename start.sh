docker build ./compiler
docker build ./runner
docker compose --env_file=.env up