# `dejavu-bq`

_"Wait.. haven't I seen this data before? ..."_

`dejavu-bq` is a small replay binary that aims to assist quick development of
bots to add to our bot army. The goal is to have something that queries bigquery
and converts rows into cloudevents a local bot can consume.

Right now in order to quickly prototype a bot, you have to stand up an obscene
amount of backing infrastructure. This small cli lets you simply plug into any
database you have access to and scrape rows and convert them to cloudevents to
start prototyping on prod data without touching a line of terraform.

It's extremely rough right now but acts as a POC of how we could do it. Please
add and make it better!

### Usage

```bash
PROJECT=prod-images-c6e5 \
    QUERY='SELECT * FROM `cloudevents_ghe_prod_rec.dev_chainguard_github_workflow_run` WHERE TIMESTAMP_TRUNC(_PARTITIONTIME, DAY) = TIMESTAMP("2024-04-18") LIMIT 5' \
    EVENT_TYPE='dev.chainguard.github.workflow_run' \
    ./dejavu-bq
```

### Developing a bot locally

The major drive behind this utility is to speed up building a bot. You can build
and run a bot locally bypassing the octosts token exchange

```bash
cd ~/path/to/my/bot

GITHUB_TOKEN=$(gh auth token) go run .
```

Now try using an octosts policy with the following command.

```bash
cd ~/path/to/my/bot

GITHUB_TOKEN=$(chainctl auth octo-sts --scope chainguard-dev/fake-repo --identity mybot) go run .
```

**NOTE:** I haven't gotten this working yet, not sure what's wrong but in theory it works
