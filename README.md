# alertreceiver

можно использовать для пересылки из обычного алертменджера в madison

```
global:
  resolve_timeout: 5m
inhibit_rules:
- equal:
  - namespace
  - alertname
  source_matchers:
  - severity = critical
  target_matchers:
  - severity =~ warning|info
- equal:
  - namespace
  - alertname
  source_matchers:
  - severity = warning
  target_matchers:
  - severity = info
- equal:
  - namespace
  source_matchers:
  - alertname = InfoInhibitor
  target_matchers:
  - severity = info
receivers:
- name: madison
  webhook_configs:
  - send_resolved: true
    url: http://alertreceiver:8080/prometheus
route:
  group_by:
  - namespace
  group_interval: 5m
  group_wait: 30s
  receiver: madison
  repeat_interval: 12h
  routes:
  - matchers:
    - severity = critical
    receiver: 'madison'
templates:
- /etc/alertmanager/config/*.tmpl

```