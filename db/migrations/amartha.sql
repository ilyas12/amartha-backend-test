/*
 Navicat Premium Dump SQL

 Source Server         : local
 Source Server Type    : MySQL
 Source Server Version : 90400 (9.4.0)
 Source Host           : localhost:3306
 Source Schema         : amartha

 Target Server Type    : MySQL
 Target Server Version : 90400 (9.4.0)
 File Encoding         : 65001

 Date: 05/09/2025 14:56:08
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for approvals
-- ----------------------------
DROP TABLE IF EXISTS `approvals`;
CREATE TABLE `approvals` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `approval_id` char(32) NOT NULL,
  `loan_id` bigint unsigned NOT NULL,
  `photo_url` text NOT NULL,
  `validator_employee_id` char(32) NOT NULL,
  `approval_date` date NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `deleted_by` char(32) DEFAULT NULL,
  `deleted_flag` tinyint(1) GENERATED ALWAYS AS (if((`deleted_at` is null),0,1)) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ux_approvals_approval_id_active` (`approval_id`,`deleted_flag`),
  UNIQUE KEY `ux_approvals_loan_active` (`loan_id`,`deleted_flag`),
  CONSTRAINT `fk_approvals_loan` FOREIGN KEY (`loan_id`) REFERENCES `loans` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for disbursements
-- ----------------------------
DROP TABLE IF EXISTS `disbursements`;
CREATE TABLE `disbursements` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `disbursement_id` char(32) NOT NULL,
  `loan_id` bigint unsigned NOT NULL,
  `signed_agreement_url` text NOT NULL,
  `signature_provider` varchar(64) NOT NULL,
  `signature_tx_id` varchar(128) NOT NULL,
  `signature_status` enum('PENDING','SIGNED','CANCELLED') NOT NULL DEFAULT 'PENDING',
  `signed_at` datetime DEFAULT NULL,
  `document_sha256` char(64) DEFAULT NULL,
  `officer_employee_id` char(32) NOT NULL,
  `disbursement_date` date NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `deleted_by` char(32) DEFAULT NULL,
  `deleted_flag` tinyint(1) GENERATED ALWAYS AS (if((`deleted_at` is null),0,1)) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ux_disb_disbursement_id_active` (`disbursement_id`,`deleted_flag`),
  UNIQUE KEY `ux_disb_loan_active` (`loan_id`,`deleted_flag`),
  UNIQUE KEY `ux_disb_sig_active` (`signature_provider`,`signature_tx_id`,`deleted_flag`),
  CONSTRAINT `fk_disb_loan` FOREIGN KEY (`loan_id`) REFERENCES `loans` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for investments
-- ----------------------------
DROP TABLE IF EXISTS `investments`;
CREATE TABLE `investments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `investment_id` char(32) NOT NULL,
  `loan_id` bigint unsigned NOT NULL,
  `investor_id` char(32) NOT NULL,
  `amount` decimal(18,2) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `deleted_by` char(32) DEFAULT NULL,
  `deleted_flag` tinyint(1) GENERATED ALWAYS AS (if((`deleted_at` is null),0,1)) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ux_investments_investment_id_active` (`investment_id`,`deleted_flag`),
  KEY `idx_investments_loan_active` (`loan_id`,`deleted_flag`),
  KEY `idx_investments_investor_active` (`investor_id`,`deleted_flag`),
  CONSTRAINT `fk_investments_loan` FOREIGN KEY (`loan_id`) REFERENCES `loans` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `investments_chk_1` CHECK ((`amount` > 0))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for loans
-- ----------------------------
DROP TABLE IF EXISTS `loans`;
CREATE TABLE `loans` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `loan_id` char(32) NOT NULL,
  `borrower_id` char(32) NOT NULL,
  `principal` decimal(18,2) NOT NULL,
  `rate` decimal(6,4) NOT NULL,
  `roi` decimal(6,4) NOT NULL,
  `agreement_link` text,
  `state` enum('proposed','approved','invested','disbursed') NOT NULL DEFAULT 'proposed',
  `state_updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `deleted_by` char(32) DEFAULT NULL,
  `deleted_flag` tinyint(1) GENERATED ALWAYS AS (if((`deleted_at` is null),0,1)) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ux_loans_loan_id_active` (`loan_id`,`deleted_flag`),
  KEY `idx_loans_borrower_active` (`borrower_id`,`deleted_flag`),
  CONSTRAINT `loans_chk_1` CHECK ((`principal` > 0))
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

SET FOREIGN_KEY_CHECKS = 1;
