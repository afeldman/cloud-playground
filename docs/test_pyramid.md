# Die Moderne Testpyramide – Von isoliert zu integriert  
_Obsidian Engineering Guide – inkl. ISO-Standards für sicheres Programmieren & Testing_

---

## 1. Grundprinzip  
Die Testpyramide basiåert nicht auf „teuer zu günstig“, sondern auf:

> **Isoliert → Teilweise integriert → Voll integriert**  
> **Deterministisch → Kontextreich → Nutzerrealität**

Kosten, Instabilität und Testdauer ergeben sich erst **als Konsequenz** zunehmender Integration.

---

## 2. Die Testpyramide (Diagramm)

```
                  ▲
                  │      4. Explorative / UX / Human Tests
                  │      (keine Isolation, maximale Realität)
     integriert   │──────────────────────────────────────────────
                  │      3. UI / End-to-End Tests
                  │      (System als Ganzes)
                  │──────────────────────────────────────────────
                  │      2. Integration / Service Tests
                  │      (mehrere Komponenten interagieren)
                  │──────────────────────────────────────────────
     isoliert     │      1. Unit Tests
                  │      (kleinste Einheiten, deterministisch)
                  ▼
```

---

# 3. Testschichten im Detail  
_Fokus: Isolationsgrad, nicht Kosten_

---

## 3.1 Unit Tests (höchste Isolation)

### Charakteristik  
- isoliert  
- keine externen Abhängigkeiten  
- extrem schnell  
- deterministisch  
- Grundlage stabiler Softwarequalität  

### Tools  
| Bereich | Tools |
|--------|-------|
| Go | testing, testify, gomock |
| Python | pytest, unittest, hypothesis |
| Terraform | validate, tflint, checkov |
| General | Mocks, Stubs |

---

## 3.2 Integration / Service Tests

### Charakteristik  
- mehrere Komponenten interagieren  
- Datenbanken, Queues, APIs  
- realitätsnäher, aber noch kontrollierbar  

### Tools  
| Bereich | Tools |
|--------|-------|
| Go | Testcontainers-Go, docker-compose |
| Python | pytest + testcontainers, FastAPI TestClient |
| Terraform | Terratest (Go), LocalStack |
| Network | WireMock, MockServer |

---

## 3.3 UI / End-to-End Tests (minimale Isolation)

### Charakteristik  
- Systemtests aus Nutzersicht  
- variabel, weniger deterministisch  
- wenige, aber kritische Tests  

### Tools  
| Bereich | Tools |
|--------|-------|
| Web UI | Playwright, Cypress, Selenium |
| Mobile | Appium, Detox |
| Visual Regression | Percy, Chromatic |
| API E2E | Karate, Newman |

---

## 3.4 Explorative / UX Tests (keine Isolation)

### Charakteristik  
- menschliche Wahrnehmung & Emotion  
- nicht deterministisch  
- evaluieren Usability, Accessibility, Experience  

### Tools  
- Maze, Hotjar, FullStory  
- Lighthouse, axe  
- Tobii Eye Tracking  
- KI-gestützte UX-Reviewer  

---

# 4. IaC-Testpyramide (Terraform)

| Ebene | Beispiel | Tools |
|-------|----------|--------|
| Unit | validate, plan-json, OPA policies | terraform validate, tflint, OPA |
| Integration | LocalStack | Terratest |
| E2E | Deployment in Test-Cloud | Terratest + Sandbox |
| Explorativ | Architektur- und Security-Review | manuell |

---

# 5. Neues Verständnis der Pyramide

| Ebene | Isolation | Integration | Determinismus | Realität |
|--------|----------|-------------|---------------|----------|
| Unit | sehr hoch | minimal | sehr stabil | gering |
| Integration | mittel | mittel | stabil | moderat |
| E2E/UI | gering | hoch | variabel | sehr hoch |
| UX | keine | vollständig | unkontrolliert | maximal |

---

# 6. ISO-Standards für sicheres Programmieren & Testing  
_Erweitern die Testpyramide um Compliance & Qualitätssicherung._

---

## 6.1 ISO/IEC 25010 – Software Quality Model  
Definiert Qualitätsattribute wie:

- Functional Suitability  
- Performance  
- Reliability  
- Security  
- Maintainability  
- Usability  

### Verbindung zur Pyramide
Unit: Korrektheit, Maintainability  
Integration: Reliability, Performance  
E2E: Functional Suitability, Compatibility  
UX: Usability  

---

## 6.2 ISO/IEC 27001 – Sicherheit in Entwicklungsprozessen  
Relevante Anforderungen:

- sichere CI/CD  
- Zugriffskontrollen  
- sichere Verarbeitung von Testdaten  
- Security-Tests als Teil der Pipeline  

### Auswirkungen
- E2E muss Sicherheitskontrollen testen  
- Testdaten müssen DSGVO-konform sein  

---

## 6.3 ISO/IEC 29119 – International Software Testing Standard  
Der umfassendste Teststandard.

Beinhaltet:

- Testprozesse  
- Testdesign  
- Testmethoden  
- Dokumentation  
- Bewertungstechniken  

### Verbindung zur Pyramide
- Unit → Whitebox  
- Integration → Interface Testing  
- E2E → Scenario-Based Testing  
- Explorativ → Experience-Based  

---

## 6.4 ISO/IEC 12207 – Software Lifecycle Processes  
Regelt die Struktur des gesamten Entwicklungsprozesses.

### Relevanz
- Tests müssen über den gesamten Lifecycle geplant sein  
- Traceability: Requirements → Testfälle → Ergebnisse  

---

## 6.5 ISO/IEC 5055 – Software Trustworthiness  
Misst:

- Security  
- Maintainability  
- Reliability  
- Technical Debt  

Besonders relevant: Unit- & Integration Tests.

---

## 6.6 ISO/IEC 20243 – Supply Chain Security  
Schützt Build- & Dependency-Herkunft.

### Relevanz
- SBOM-Pflicht  
- Dependency-Scanning  
- reproduzierbare Builds  

---

## 6.7 ISO/SAE 21434 – Secure Software Engineering  
Auch außerhalb Automotive einsetzbar:

- Threat Modeling  
- Security Verification  
- Continuous Cybersecurity Testing  

---

# 7. Secure Coding Anforderungen (ISO + CERT + OWASP)

- Eingaben validieren  
- Keine Secrets im Code  
- Starkes Typensystem verwenden  
- Sichere Fehlerbehandlung  
- Logging ohne personenbezogene Daten  
- Least Privilege überall  
- Reproduzierbares Bauen sicherstellen  
- IaC signieren & versionieren  

### Tools  
| Sprache | Tools |
|---------|-------|
| Go | gosec, govulncheck, staticcheck |
| Python | bandit, safety, pip-audit |
| Terraform | tfsec, checkov, opa |
| Web | OWASP ZAP, Burp Suite |

---

# 8. Kombinierte Testpyramide inkl. ISO-Bezug

```
                        ▲
                        │      4. UX / Explorativ
                        │      ISO 25010 (Usability)
                        │      ISO 29119-5 (Experience-Based)
   vollständige         │────────────────────────────────────────────
   Integration          │      3. End-to-End / UI Tests
                        │      ISO 29119 / ISO 12207
                        │      ISO 27001 (Security)
                        │────────────────────────────────────────────
   partielle            │      2. Integration / Service Tests
   Integration          │      ISO 29119-4 (Techniques)
                        │      ISO 20243 (Supply Chain)
                        │────────────────────────────────────────────
   hohe Isolation       │      1. Unit Tests
                        │      ISO 5055 (Maintainability)
                        │      Secure Coding Standards
                        ▼
```

---

# 9. Quellen  
- Martin Fowler – The Practical Test Pyramid  
- OnPathTesting – Agile Testing Pyramid  
- Go Testing Demystified  
- ISO/IEC 25010, 27001, 29119, 12207, 5055, 20243, 21434  

---

Fertig. Diese Version ist für Obsidian optimiert (Header, Struktur, Markdown-Kompatibilität).


---

# 10. Environments & Delivery Stages: Playground → Dev → Stage → Prod  
_Ergänzung der Testpyramide um reale Deployment-Umgebungen_

Die Testpyramide beschreibt **Testarten**, aber in der Praxis müssen diese in unterschiedlichen **Umgebungen** ausgeführt werden.  
Moderne Unternehmen (z. B. Birdy) nutzen mindestens vier Umgebungen:

```
Playground → Dev → Stage → Prod
```

Jede Umgebung hat einen anderen Zweck, andere Stabilitätsanforderungen und unterschiedliche Testschwerpunkte.

---

## 10.1 Übersicht der Umgebungen

| Umgebung | Zweck | Stabilität | Risiko | Haupt-Testarten |
|----------|--------|-----------|--------|------------------|
| **Playground** | Experimentieren, Prototypen, schnelle Iteration | sehr niedrig | hoch | Unit, schnelle Integration |
| **Dev** | aktives Entwickler-Testen, Feature Branch Deployments | mittel | mittel | Unit, Integration, API-Tests |
| **Stage** | Vorproduktion, reale Datenstrukturen, End-to-End | hoch | niedrig | Integration, E2E, Security |
| **Prod** | Live-Betrieb, Kundenverkehr | sehr hoch | kritisch | Monitoring, Canary Tests, Synthetics |

---

## 10.2 Beschreibung der Umgebungen im Detail

### **Playground (Experimentierumgebung)**  
Zweck:
- neue Ideen ausprobieren  
- schnelle Prototypen  
- KI-Assistenten-Code testen  
- kein Fokus auf Stabilität

Typische Tests:
- spontane Unit-Tests  
- Mock-basierte Integration  
- experimentelle Tools (z. B. Fuzzing)

Tools:
- local docker-compose  
- Testcontainers  
- Feature Flags

---

### **Dev (Entwicklungsumgebung)**  
Zweck:
- Feature Branch Deployments  
- Teaminternes Testen  
- frühzeitige Integration

Testschwerpunkte:
- Unit  
- Integration  
- erste API-Tests  
- Linting, Static Analysis

Sicherheitsrelevante Checks (ISO 27001):
- Secrets-Scanning  
- Dependency-Audits  
- Least Privilege in CI/CD  

---

### **Stage (Vorproduktion / Pre-Prod)**  
Zweck:
- Produktionsähnliche Umgebung  
- E2E-Flows  
- Lasttests  
- Sicherheitsvalidierung

Testschwerpunkte:
- E2E UI Tests (Playwright, Cypress)  
- Security Tests (OWASP ZAP, Burp)  
- Performance (k6, Locust)  
- Integration mit echten Cloud Services

ISO-Bezug:
- ISO 29119 (formale Tests)  
- ISO 12207 (Lifecycle)  
- ISO 27001 (Security Verification)

Stage ist die letzte Barriere vor Prod.

---

### **Prod (Produktion)**  
Zweck:
- realer Kundenverkehr  
- hohe Stabilität  
- Monitoring statt klassischem Testing  

Testschwerpunkte:
- synthetische Tests (Blackbox Monitoring)  
- Canary Releases  
- Chaos Engineering (optional)  
- Alerting & Observability  

ISO-Bezug:
- ISO 27001 Logging/Monitoring  
- ISO 25010 Reliability  
- ISO 5055 Trustworthiness  

---

# 11. Verbindung: Testpyramide × ISO × Deployment-Stages

Die drei Modelle zusammen ergeben folgendes Bild:

```
                           ▲ Prod
                           │  (Monitoring, Synthetics, Reliability)
                           │────────────────────────────────────────────
                           │ Stage
                           │  (E2E, Security, Load, Compliance)
   zunehmende Integration  │────────────────────────────────────────────
                           │ Dev
                           │  (Integration, API, Static Analysis)
                           │────────────────────────────────────────────
                           │ Playground
   höchste Isolation       │  (Unit, Experimentelles Testing)
                           ▼
```

Matrix:

| Testart | Isolationsgrad | Playground | Dev | Stage | Prod |
|---------|----------------|-----------|-----|--------|-------|
| Unit | hoch | ✔ | ✔ | optional | selten |
| Integration | mittel | ✔ | ✔✔ | ✔ | selten |
| E2E | gering | selten | teilweise | ✔✔ | synthetisch |
| UX/Explorativ | keine | Experimente | interne Reviews | externe Tests | Feedback |
| Security | je nach Layer | Tools testen | CI/CD Security | Pentests / ZAP | Monitoring |
| IaC Tests | hoch–mittel | validate | integration | staged apply | prod-guardrails |

---

# 12. Finaler Überblick – das vollständige Modell

```
 Isolation ↓          Testarten →     Unit → Integration → E2E → UX/Security
──────────────────────────────────────────────────────────────────────────────
Stage: Playground      ✔✔✔            ✔             (experimentell)      –
Stage: Dev             ✔✔             ✔✔            ✔ (ausgewählte)      –
Stage: Stage           optional        ✔✔           ✔✔✔                  ✔
Stage: Prod            selten          selten        synthetisch          Feedback
──────────────────────────────────────────────────────────────────────────────
ISO-Bezug:           5055/25010      29119/20243   29119/12207/27001    25010/29119
```

---

# 13. Empfehlung für cloud-playground Engineering Workflow

- **Playground (dieser Repo):** Kind + LocalStack, KI-Experimentierfläche via MCP
- **Dev:** Feature Branch Deployments, automatisierte Unit + Integration
- **Stage:** End-to-End & Security Gates
- **Prod:** Canary Deployment + Synthetische Monitoring-Tests

Dieses Modell ist stabil, ISO-kompatibel und skalierbar.

---

