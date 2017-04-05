/*
 * Copyright (c) SAS Institute Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package token

import (
	"crypto"
	"fmt"
	"path/filepath"

	"gerrit-pdt.unx.sas.com/tools/relic.git/cmdline/shared"
	"gerrit-pdt.unx.sas.com/tools/relic.git/config"
	"gerrit-pdt.unx.sas.com/tools/relic.git/lib/audit"
	"gerrit-pdt.unx.sas.com/tools/relic.git/lib/certloader"
	"gerrit-pdt.unx.sas.com/tools/relic.git/p11token"
)

func NewAudit(key *p11token.Key, sigType string, hash crypto.Hash) *audit.AuditInfo {
	info := audit.New(key.Name, sigType, hash)
	if argFile != "" && argFile != "-" && info.Attributes["client.filename"] == nil {
		info.Attributes["client.filename"] = filepath.Base(argFile)
	}
	return info
}

func PublishAudit(info *audit.AuditInfo) error {
	if err := shared.InitConfig(); err != nil {
		return err
	}
	aconf := shared.CurrentConfig.Amqp
	if aconf != nil && aconf.Url != "" {
		if aconf.SealingKey != "" {
			if err := sealAudit(info, aconf); err != nil {
				return shared.Fail(fmt.Errorf("failed to seal audit log: %s", err))
			}
		}
		if err := info.Publish(aconf); err != nil {
			return shared.Fail(fmt.Errorf("failed to publish audit log: %s", err))
		}
	}
	if err := info.WriteFd(); err != nil {
		return shared.Fail(fmt.Errorf("failed to publish audit log: %s", err))
	}
	return nil
}

func sealAudit(info *audit.AuditInfo, aconf *config.AmqpConfig) error {
	key, err := openKey(aconf.SealingKey)
	if err != nil {
		return err
	}
	cert, err := certloader.LoadTokenCertificates(key, key.X509Certificate, "")
	if err != nil {
		return err
	}
	return info.Seal(cert.Signer(), cert.Chain(), crypto.SHA256)
}
