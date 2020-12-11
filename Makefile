run: cleanup
	mkdir -p past-runs
	cp startup.log "./past-runs/startup-$$(date +%s).log"
	go run main.go

cleanup:
	kubectl delete dw timing-test -n timing-test || true

process:
	@ { \
		echo "numContainers,totalTime (ms),componentsTime (ms),routingTime (ms),deploymentTime (ms),healthChecksTime (ms)" ;\
		jq -r -f summarize.jq startup.log \
		| sed 's| ms||g' ;\
	}
