---
title: Deploying
description: From a release tag to the server.
---

Production is one Ubuntu server running `compose.prod.yml`, next to StoreIt, behind the host nginx.

## First install

```sh
git clone <repo> /opt/sunshine-forum && cd /opt/sunshine-forum
sh deploy/scripts/init.sh            # deploy/secrets/: Postgres password, privacy keys
echo -n "<resend key>" > deploy/secrets/resend_api_key.txt
cp .env.prod.example .env            # set APP_URL, MAIL_FROM, ADMIN_EMAIL, ADMIN_PASSWORD
```

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

## Each release

A release is a `vX.Y.Z` tag on `main`, after the CI checks pass.

```sh
cd /opt/sunshine-forum
git fetch --tags && git checkout vX.Y.Z
make images TAG=vX.Y.Z
TAG=vX.Y.Z docker compose -f compose.prod.yml up -d    # migrate runs before app and worker
```

`migrate` applies new goose and River migrations and exits; `app` and `worker` start only after it succeeds.
Migrations must stay backward compatible for one release, because the old `app` keeps serving until the new one
is healthy.

After the first start, remove `ADMIN_PASSWORD` from `.env`; the account already exists.

## Rolling back

Check out the previous tag and run the same two commands. Migrations are not rolled back automatically; write a new
forward migration instead.
