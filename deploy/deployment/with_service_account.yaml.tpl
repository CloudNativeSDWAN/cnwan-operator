# NOTE: this is a template file and it is not meant to be deployed as-is.
# Please use /scripts/deploy.sh instead of deploying this file.
# Alternatively, please replace all necessary variables, defined as {VAR}
# prior to deploying this.
kind: Deployment
apiVersion: apps/v1
metadata:
  name: cnwan-operator-controller-manager
  namespace: cnwan-operator-system
  labels:
    control-plane: controller-manager
    cnwan.io/application: operator
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
      cnwan.io/application: operator
  template:
    metadata:
      labels:
        control-plane: controller-manager
        cnwan.io/application: operator
    spec:
      volumes:
        - name: settings
          configMap:
            name: cnwan-operator-settings
        - name: service-handler-secret
          secret:
            secretName: cnwan-operator-service-handler-account
      containers:
        - name: manager
          image: {CONTAINER_IMAGE}
          resources:
            limits:
              cpu: 100m
              memory: 30Mi
            requests:
              cpu: 100m
              memory: 20Mi
          volumeMounts:
            - name: settings
              readOnly: true
              mountPath: /settings
            - name: service-handler-secret
              readOnly: true
              mountPath: /credentials
          imagePullPolicy: Always
          env:
          - name: CNWAN_OPERATOR_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
      restartPolicy: Always
      serviceAccountName: cnwan-operator-service-account
      serviceAccount: cnwan-operator-service-account