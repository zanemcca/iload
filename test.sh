export TUTUM_USER=zanemcca
export TUTUM_PASS=@1Ph@c3p0f!sh

function quit {
  # Clean up the stack
  tutum stack terminate --sync testStack
  exit $1
}

zeroDone=1 
# End the tryZeroDown call
function zeroDone {
  zeronDone=0
}

# Continuosly curl the load balancer and never expect anything but 200
function tryZeroDown {
  failed=0
  count=0
  while $zeroDone > 0] 
  do
    count = $count + 1
    if [ curl --retry 10 --retry-delay 5 -sL -w "%{http_code}\\n" \
      "http://load.testStack.`echo $TUTUM_USER`.svc.tutum.io" -o /dev/null \
      -ne 200 ]
    then
      failed = $failed + 1 
    fi
    sleep 1
  done
  if [ $failed > 0 ]
  then
    echo "FAILED: There were $failed non '200' response codes recieved of $count total calls"
    quit -1
  else
    echo zero down!
  fi
  zeroDone=1 
}

# Try will send a curl request to the load balancer and then verify that the proper
#   hostnames are returned 
# try n 
# Where n is the number of active instances that are to be balanced across
function try {

  sleep 30 
  rm output
  # Perform $1 curl requests and log the output
  for i in `seq 1 $((10 * $1))`;
  do
    curl --retry 10 --retry-delay 5 -sSfL http://load.testStack.`echo $TUTUM_USER`.svc.tutum.io >> output 
  done

  # Check that $1 app instances were successfully load balanced across
  for i in `seq 1 $1`;
  do
    if grep -Fq "My hostname is app-$i"  output
    then
      echo "Success: app-$i was called!"
    else
      echo "FAILED: app-$i was not called!"
      quit -1 
    fi
  done
}

# Setup the image and login to ccounts/login/tutum
tutum login -u `echo $TUTUM_USER`  -p `echo $TUTUM_PASS` -e test@instanews.com 
docker build -t iload .
tutum image push iload

# Start the stack and wait for it to be running before continuing
tutum stack up --sync -n testStack -f tutum.yml
#tutum service run -t 2 --name webTest tutum/hello-world
#tutum service run --role global --link webTest:webTest --name iloadTest tutum.co/`echo $TUTUM_USER`/iload 

echo "===> Testing that all 3 instances of the app are being used"
try 3

echo "===> Testing scaling up command on the app service"
tutum service scale --sync app 4
try 4

echo "===> Testing scaling down command on the app service"
tutum service scale --sync app 3 
try 3

echo "===> Testing stopping a container of the app service"
tutum container stop --sync app-3
try 2

echo "===> Testing starting a container of the app service"
tutum container start --sync app-3
try 3 

echo "===> Testing terminating a container of the app service"
tutum container terminate --sync app-3
try 2

echo "===> Testing zero-downtime redeployment"
tryZeroDown &

echo "======> Load Balancer reload"
tutum service redeploy --sync load

echo "======> App service reload"
tutum service redeploy --sync app 

echo "======> Reload the whole stack"
tutum stack redeploy --sync testStack 

zeroDone

quit
