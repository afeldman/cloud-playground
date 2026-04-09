package sanitize

import "log/slog"

func sanitizeObject(obj map[string]interface{}) bool {
	kind, _ := obj["kind"].(string)

	// 🚫 NEVER allow Jobs
	if kind == "Job" {
		return false
	}

	// 🚫 Remove default ServiceAccount
	if kind == "ServiceAccount" {
		if meta, ok := obj["metadata"].(map[string]interface{}); ok {
			if meta["name"] == "default" {
				return false
			}
		}
	}

	// 🚫 kube-root-ca.crt is cluster-generated
	if kind == "ConfigMap" {
		if meta, ok := obj["metadata"].(map[string]interface{}); ok {
			if meta["name"] == "kube-root-ca.crt" {
				return false
			}
		}
	}

	// ✅ Allowlist
	allowed := map[string]bool{
		"Deployment":  true,
		"Service":     true,
		"CronJob":     true,
		"ConfigMap":   true,
		"Secret":      true,
		"Ingress":     true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"HPA":         true,
		"Role":        true,
		"RoleBinding": true,
	}

	if !allowed[kind] {
		slog.Info("unsupported kind", "kind", kind)
		return false
	}

	// ─────────────────────────────────────────
	// Metadata cleanup
	// ─────────────────────────────────────────
	if meta, ok := obj["metadata"].(map[string]interface{}); ok {
		delete(meta, "uid")
		delete(meta, "resourceVersion")
		delete(meta, "creationTimestamp")
		delete(meta, "generation")
	}

	// status entfernen
	delete(obj, "status")

	spec, _ := obj["spec"].(map[string]interface{})

	switch kind {

	case "Service":
		delete(spec, "clusterIP")
		delete(spec, "clusterIPs")

	case "CronJob":
		jobTemplate, ok := spec["jobTemplate"].(map[string]interface{})
		if !ok {
			return true
		}

		jobSpec, ok := jobTemplate["spec"].(map[string]interface{})
		if !ok {
			return true
		}

		delete(jobSpec, "selector")

	case "Deployment":
		if tmpl, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := tmpl["spec"].(map[string]interface{}); ok {
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					for _, c := range containers {
						container, ok := c.(map[string]interface{})
						if !ok {
							continue
						}

						// 🔥 lifecycle.preStop verursacht Invalid-Errors → immer entfernen
						delete(container, "lifecycle")
					}
				}
			}
		}
	}

	return true
}
