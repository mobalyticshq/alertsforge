# alertsforge
magic labels:
alertsforge_delay_resolve: 20h
alertsforge_escalation_chain: devops

***

curl command example to post test alert to alertsforge buffer
curl --location 'http://127.0.0.1:8080/alertWebhook/api/v2/alerts' \
--header 'Content-Type: application/json' \
--data '[    {
            "labels": {
                "alertgroup": "oom-alerts",
                "alertname": "container-oom",
                "alertsforge_title": "test alertgroup",
                "channel": "alerts-p2",
                "cluster": "app",
                "instance": "10.1.1.1:8080",
                "test": "test",
                "namespace": "prod",
                "pod": "app",
                "container": "app",
                "severity": "p2"
            },
            "annotations": {
                "description": "test! container restart",
                "summary": "test! restart"
            },
            "startsAt": "2024-06-05T19:10:47.047759595Z",
            "endsAt": "2024-06-05T19:12:47.047759595Z"
}
]'