/*
 * This file is part of the kiagnose project
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
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package config

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kiagnose/kiagnose/kiagnose/internal/configmap"
	"github.com/kiagnose/kiagnose/kiagnose/internal/rbac"
)

const (
	ConfigMapNamespaceEnvVarName = "CONFIGMAP_NAMESPACE"
	ConfigMapNameEnvVarName      = "CONFIGMAP_NAME"
)

type Loader struct {
	client kubernetes.Interface
	env    map[string]string
}

func NewLoader(client kubernetes.Interface, env map[string]string) *Loader {
	return &Loader{client: client, env: env}
}

func (l *Loader) Load() (*Config, error) {
	const envVarErr = "missing required environment variable"

	configMapNamespace, exists := l.env[ConfigMapNamespaceEnvVarName]
	if !exists {
		return nil, fmt.Errorf("%s: %q", envVarErr, ConfigMapNamespaceEnvVarName)
	}

	configMapName, exists := l.env[ConfigMapNameEnvVarName]
	if !exists {
		return nil, fmt.Errorf("%s: %q", envVarErr, ConfigMapNameEnvVarName)
	}

	rawData, err := configmap.GetData(l.client, configMapNamespace, configMapName)
	if err != nil {
		return nil, err
	}

	parser := newConfigMapParser(rawData)
	err = parser.Parse()
	if err != nil {
		return nil, err
	}

	clusterRoles, err := rbac.GetClusterRoles(l.client, parser.ClusterRoleNames())
	if err != nil {
		return nil, err
	}

	roles, err := rbac.GetRoles(l.client, parser.RoleNames())
	if err != nil {
		return nil, err
	}

	return &Config{
		Image:        parser.Image(),
		Timeout:      parser.Timeout(),
		EnvVars:      paramsToEnvVars(parser.Params()),
		ClusterRoles: clusterRoles,
		Roles:        roles,
	}, nil
}

func paramsToEnvVars(params map[string]string) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	for k, v := range params {
		envVars = append(envVars, corev1.EnvVar{
			Name:  strings.ToUpper(k),
			Value: v,
		})
	}

	return envVars
}
