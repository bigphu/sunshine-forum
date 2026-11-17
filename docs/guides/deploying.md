---
title: Deploying
description: From a release tag to the server.
---

Production is one Ubuntu server running `compose.prod.yml`, next to StoreIt, behind the host nginx.

## First install

```sh
git clone <repo> /opt/sunshine-forum && cd /opt/sunshine-forum   # then fetch model/ as below
sh deploy/scripts/init.sh            # deploy/secrets/: Postgres password, privacy keys, empty Resend key file
echo -n "<resend key>" > deploy/secrets/resend_api_key.txt
cp .env.prod.example .env            # set APP_URL, MAIL_FROM, ADMIN_EMAIL, ADMIN_PASSWORD
```

`init.sh` leaves `deploy/secrets/` at mode 700 and the files at 644. The backend containers run as a non-root user
(uid 65532) and compose mounts secret files with their host permissions, so files at 600 make `migrate`, `app`
and `worker` exit at start-up. The 700 directory still keeps other users on the server out. If you create a secret
file by hand, `chmod 644` it.

**The model.** The trained model is not in git. Train it on a machine with ViHSD access (`model/README.md`), copy
`model/models/svm_final.joblib` to the same path on the server, then build its image; use the model's version as
the tag:

```sh
make model-image MODEL_TAG=svm-2026.11.17
```

and set `MODEL_TAG=svm-2026.11.17` in `.env`. To run without a model for now, set `SCREENING_MODEL_URL=` (empty)
instead: the backend falls back to its stand-in with sample scores.

Back up `deploy/secrets/privacy_enc_key.txt` and `privacy_hmac_key.txt` somewhere safe. Without them the
encrypted names, emails and units cannot be read, and nobody can sign in.

Install the host site. Add an A record for the name in Cloudflare (set to "DNS only"). The hosting provider blocks
inbound traffic from abroad, so Let's Encrypt cannot reach port 80: issue the certificate with the DNS challenge,
reusing StoreIt's Cloudflare credentials, then enable the site:

```sh
sudo certbot certonly --dns-cloudflare --dns-cloudflare-credentials /etc/letsencrypt/cloudflare.ini -d sunshine-forum.bigphu.io.vn --deploy-hook "systemctl reload nginx"
sudo cp deploy/nginx/sunshine-forum.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/sunshine-forum.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

Until the site is enabled, browsers get StoreIt's certificate for this name and show a security warning. The site proxies to the `web` container
on `127.0.0.1:8092`, with buffering turned off for the notification stream.

## Access to the model repo

`model/` is a private repository (a fork of the model team's private repo), so neither the forum's CI nor the
server can fetch it anonymously. Both use read-only credentials.

**CI.** Create a fine-grained personal access token (GitHub → Settings → Developer settings → Fine-grained
tokens) with access to `bigphu/vietnamese-toxic-comment-detection` only, permission *Contents: Read-only*, and an
expiry. Store it as the forum repo's `MODEL_REPO_TOKEN` secret:

```sh
gh secret set MODEL_REPO_TOKEN -R bigphu/sunshine-forum    # paste the token when asked
```

The `model / test` job uses it to check out the submodule. Renew it before it expires.

**Server.** Use a read-only deploy key that only this server holds:

```sh
ssh-keygen -t ed25519 -f ~/.ssh/sunshine_model -N "" -C "sunshine-forum server"
cat ~/.ssh/sunshine_model.pub      # add on GitHub: model repo → Settings → Deploy keys, read-only
cat >> ~/.ssh/config <<'EOF'
Host github-sunshine-model
  HostName github.com
  User git
  IdentityFile ~/.ssh/sunshine_model
  IdentitiesOnly yes
EOF
cd /opt/sunshine-forum
git submodule init
git config submodule.model.url git@github-sunshine-model:bigphu/vietnamese-toxic-comment-detection.git
git submodule update
```

The URL override lives in this clone's `.git/config` only, so later `git submodule update` calls keep using the key.

## Each release

A release is a `vX.Y.Z` tag on `main`, after the CI checks pass.

```sh
cd /opt/sunshine-forum
git fetch --tags && git checkout vX.Y.Z && git submodule update --init
make images TAG=vX.Y.Z
TAG=vX.Y.Z docker compose -f compose.prod.yml up -d    # migrate runs before app and worker
```

`migrate` applies new goose and River migrations and exits; `app` and `worker` start only after it succeeds.
Migrations must stay backward compatible for one release, because the old `app` keeps serving until the new one
is healthy.

After the first start, remove `ADMIN_PASSWORD` from `.env`; the account already exists.

**A new model** is its own release: build it with `make model-image MODEL_TAG=<new version>`, set `MODEL_TAG` in
`.env`, and run `docker compose -f compose.prod.yml up -d model`. While the container restarts the forum applies
its outage policy and rescans held content once the model answers, so swap at a quiet time.

## Rolling back

Check out the previous tag and run the same two commands. Migrations are not rolled back automatically; write a new
forward migration instead.
