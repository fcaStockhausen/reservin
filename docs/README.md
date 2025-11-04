# Documentation - Reservas Calculator

## Documentation Architecture

This repository follows a consolidated documentation approach with **4 source of truth documents** to avoid fragmentation and ensure clarity.

## Source of Truth Documents

### 📋 CRUSH.md
**Purpose:** Agent development guide and project overview
**Contents:** Technology stack, build commands, Git workflow, professional standards, implementation planning

### 🛠️ technical_specifications.md
**Purpose:** Complete technical implementation requirements
**Contents:** Mathematical formulas, database schema, performance targets, implementation architecture

### 📜 normative_framework.md  
**Purpose:** CMF regulatory framework and compliance requirements
**Contents:** NCG 318 analysis, missing documents, policy application rules, compliance testing

### 🚀 deployment_guide.md
**Purpose:** Production deployment and operations
**Contents:** Build process, installation scripts, service management, monitoring, troubleshooting

### 📝 implementation_plan.md
**Purpose:** Complete 8-week implementation roadmap
**Contents:** Phase-by-phase development plan, technical details, risk management, success criteria

## Supporting Analysis

### 📁 analysis/
**Purpose:** Temporary analysis documents for reference during development
**Contents:** Detailed Excel analysis, missing references research, implementation notes

### 📁 normativo/
**Purpose:** Regulatory documents and summaries for quick reference
**Contents:** PDF documents and generated summaries with key regulatory terms

**Note:** These documents provide detailed background information but should not be used as primary references. Content should be consolidated into 4 source of truth documents as needed.

## Quick Reference

| Need | Document | Section |
|------|----------|---------|
| **Implementation plan** | implementation_plan.md | Complete 8-week roadmap |
| **Development setup** | CRUSH.md | Development Environment |
| **Mathematical formulas** | technical_specifications.md | Core Mathematical Formulas |
| **CMF compliance** | normative_framework.md | Core Regulatory Requirements |
| **VTD vector tasa de descuento** | technical_specifications.md | Discount Rate Methodology |
| **Causante/Beneficiario tables** | normative_framework.md | Mortality Table Requirements |
| **Build and deploy** | deployment_guide.md | Build Process |
| **Database schema** | technical_specifications.md | Database Schema |
| **Performance targets** | technical_specifications.md | Performance Requirements |
| **Installation** | deployment_guide.md | Installation Scripts |

## Document Philosophy

1. **Single Source of Truth:** Each domain has one authoritative document
2. **No Fragmentation:** Related information is consolidated, not scattered
3. **Clear Purpose:** Each document has a well-defined scope
4. **Professional Standards:** Scientific/regulatory tone throughout
5. **Maintainability:** Easy to update and reference during development

## Usage Guidelines

### For Development
- **Primary References:** CRUSH.md + technical_specifications.md
- **Compliance Checks:** normative_framework.md
- **Deployment:** deployment_guide.md

### For Regulatory Review
- **Primary Reference:** normative_framework.md
- **Technical Details:** technical_specifications.md

### For Operations
- **Primary Reference:** deployment_guide.md
- **Troubleshooting:** technical_specifications.md

This architecture ensures clear, maintainable documentation that serves all project stakeholders without information fragmentation.