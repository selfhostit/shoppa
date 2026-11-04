# Shoppa on a Synology NAS

This guide sets Shoppa up on a Synology DS220+ for a household: the app on the NAS,
and both phones on `https://shoppa.<name>.synology.me`, at home and in the shop. It
takes about twenty minutes, and the first product import runs by itself afterwards.

```
 phones ──HTTPS──▶ router :443 ──▶ DSM reverse proxy ──▶ Shoppa on 127.0.0.1:8094 ── /volume1/docker/shoppa/pb_data
                                                              │
                                     Open Food Facts ◀────────┘  monthly export, product photos once
```

`<name>` is your Synology DDNS name, the same one the other apps on the NAS use.
Shoppa uses port 8094, next to Splitwiz (8090), Flowy (8091), Troye (8092) and should-we (8093).

## Before you start

- DSM 7.2 or later, with **Container Manager** installed from Package Center.
- A DDNS name under Control Panel > External Access > DDNS, with a Let's Encrypt
  wildcard certificate for `*.<name>.synology.me` (Control Panel > Security >
  Certificate). If the other apps on the NAS already use `something.<name>.synology.me`,
  you have both.
- Port 443 on the router forwarded to the NAS.
- SSH on: Control Panel > Terminal & SNMP > Enable SSH service.

## 1. Install the app

Sign in to the NAS from a computer: `ssh <your DSM user>@<NAS address>`. Then run:

```sh
curl -fsSL https://raw.githubusercontent.com/selfhostit/shoppa/main/deploy/install.sh |
  sudo PORT=8094 sh -s /volume1/docker/shoppa
```

Press Enter when it asks for a Tailscale key; the reverse proxy below replaces it.
The script creates `/volume1/docker/shoppa` with the settings in `.env` and the
database in `pb_data`, owned by your DSM user, and starts the app. It ends with
`shoppa is running`.

## 2. Settings

Set your time zone, so the monthly product import and the nightly backup run at
night:

```sh
cd /volume1/docker/shoppa
sudo sed -i 's/^TZ=.*/TZ=Europe\/Brussels/' .env
sudo docker compose up -d
```

## 3. The web address

1. Control Panel > Login Portal > Advanced > **Reverse Proxy** > Create:
   - Name: `shoppa`
   - Source: protocol **HTTPS**, host name `shoppa.<name>.synology.me`, port **443**,
     and tick **Enable HSTS**.
   - Destination: protocol **HTTP**, host name `localhost`, port **8094**.
2. Control Panel > Security > Certificate > **Settings**: set the wildcard certificate
   for `shoppa.<name>.synology.me`.
3. Open `https://shoppa.<name>.synology.me` on a computer. You should see the setup
   page.

Push notifications and installing the app on a phone need this HTTPS address.

## 4. Your accounts

1. On the setup page, create your own account. The first account is the owner, and
   the same email and password open the admin page at `/_/`. Use a strong password,
   since the admin page is reachable from the internet too.
2. Add the other person: open `https://shoppa.<name>.synology.me/_/`, go to
   Collections > **users** > New record, and fill in their email, name and a starting
   password (twice), with **role** set to `member`. Save.
3. Nobody else can create an account: after the owner, only the owner can add people.

Everyone signed in shares the same lists. A **Groceries** list is there to start with.

## 5. The products

A minute after the first start, the NAS starts reading the Open Food Facts export
(about a gigabyte, streamed, so it needs no extra disk) and keeps the Belgian and
Dutch products. It takes a while on a DS220+; until it is done, everyday items and
typed items work, and a barcode scan looks the product up live. After that it runs
on the 10th of each month at 04:30, a week after Troye's. The admin page shows how
it went, under Collections > **product_imports**.

## 6. On each phone

1. Open `https://shoppa.<name>.synology.me` and sign in: in **Chrome** on Android,
   in **Safari** on an iPhone.
2. Install it: Chrome's menu (⋮) > **Install app** (or **Add to home screen**); on an
   iPhone, Share > **Add to Home Screen**. Shoppa now opens from its own icon, full
   screen.
3. In the **Me** tab, pick the language item names show in, and turn on
   notifications. That is how the phone hears "Anna added 3 items to Groceries".
   On an iPhone, notifications only work from the installed app.
4. Try it: add "melk" on one phone and watch it appear on the other.

## Backups

The app backs up its database and product photos every night at 03:00 and keeps the
last 7, in `pb_data/backups` on the same disk. To keep a copy elsewhere, add
`docker/shoppa` to a **Hyper Backup** task, to a USB disk or another NAS. To restore
a nightly backup, use the admin page: Settings > Backups.

## Updates

```sh
cd /volume1/docker/shoppa && sudo docker compose pull && sudo docker compose up -d
```

Running the install script again does the same and also refreshes the compose
files. Your settings and data stay.

## When something is off

- **See the logs:** Container Manager > Container > the `shoppa-app-1` container >
  Log, or `sudo docker compose logs app` in the folder.
- **The page doesn't load:** check that `curl http://127.0.0.1:8094/api/health`
  answers on the NAS, then the reverse proxy rule and the certificate.
- **The install stops with a port error:** another app has the port. Set `PORT` in
  `.env` to a free one, run `sudo docker compose up -d`, and change the reverse
  proxy's destination port to match.
- **No notifications:** open the app from the HTTPS address (not the NAS's IP),
  turn them on in the Me tab, and check that the phone lets the browser show
  notifications.
- **No products in search:** the first import may still be running; see
  **product_imports** in the admin page.
