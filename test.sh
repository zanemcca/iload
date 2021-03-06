TUTUM_USER=zanemcca
TUTUM_PASS="@1Ph@c3p0f!sh"

RANDOM=$$
RAND=$RANDOM
STACKNAME=$RAND-testStack
LOADNAME=$RAND-load
APPNAME=$RAND-app
URL="http://`echo $LOADNAME`.`echo $STACKNAME`.`echo $TUTUM_USER`.svc.tutum.io"
echo $URL

##################################################################################
# Functions 
##################################################################################

# Clean up function
function quit {
  kill -15 $ZERO_PID
  rm -f custom.yml
  rm -f output
  # Clean up the stack
  tutum stack terminate $STACKNAME 
  exit $1
}

# Catch signals and run cleanup function
trap quit SIGHUP SIGINT SIGTERM

# Get the httpCode from the URL
function getHttpCode {
  return `curl --retry 10 --retry-delay 5 -sL -w "%{http_code}\\n" $URL -o /dev/null`
}

# Continuosly curl the load balancer and never expect anything but 200
function tryZeroDown {

  local failed=0
  local count=0 

  trap "zeroClean \$failed \$count" SIGHUP SIGINT SIGTERM

  while : 
  do
    count=$(($count + 1))
    if getHttpCode != 200
    then
      failed=$(($failed + 1)) 
    fi
    sleep 1
  done
}

# Clean up function for the tryZeroDown function
# zeroClean $failed $count
function zeroClean {
  if (($1 > 0));
  then
    echo "FAILED: There were $1 non '200' response codes recieved of $2 total calls"
    exit -1
  else
    echo "Success: All $2 requests returned 200 as expected" 
    exit
  fi
}

# Try will send a curl request to the load balancer and then verify that the proper
#   hostnames are returned 
# try n 
# Where n is the number of active instances that are to be balanced across
# n < 10
function try {

  local prev=0
  local delay=1

  local failed=0

  while (($delay < 55));
  do
    rm -f output
    # Perform $1 curl requests and log the output
    for i in `seq 1 $((10 * $1))`;
    do
      curl --retry 10 --retry-delay 5 -sSfL $URL >> output
    done

    failed=0
    # Check that $1 app instances were successfully load balanced across
    for i in `seq 1 $1`;
    do
      if ! grep -q "My hostname is .*-app-$i" output;
      then
        failed=$i
        break
      fi
    done

    # Check that 10-$1 app instances are not being load balanced
    if (($1 < 10));
    then
      for i in `seq $(($1+1)) 10`;
      do
        if grep -q "My hostname is .*-app-$i" output;
        then
          failed=$i
          break
        fi
      done
    fi

    # If the test failed then increase the delay and sleep
    if (($failed > 0));
    then
      echo "Trying again in $delay seconds"
      sleep $delay
      delay=$(($delay + $prev))
      prev=$(($delay - $prev))
    else
        echo "Success: All $1 apps were called!"
        break
    fi

  done

  # Check if the test has failed
  if (($failed > 0));
  then
    echo "FAILED: app-$failed was not called!"
    quit -1
  fi
}

##################################################################################
# Main
##################################################################################

# Setup the image and login to ccounts/login/tutum
tutum login -u $TUTUM_USER  -p $TUTUM_PASS -e zane@instanews.com 
docker build -t iload .
tutum image push iload

sed "s/lb/$LOADNAME/g" tutum.yml > custom.yml
sed -i "s/app/$APPNAME/g" custom.yml
sed -i "s/username/$TUTUM_USER/g" custom.yml

# Start the stack and wait for it to be running before continuing
tutum stack up --sync -n $STACKNAME -f custom.yml

echo "===> Testing that all 3 instances of the app are being used"
try 3
echo

echo "===> Testing scaling up command on the app service"
tutum service scale --sync $APPNAME 4 >> /dev/null
try 4
echo

echo "===> Testing scaling down command on the app service"
tutum service scale --sync $APPNAME 3 >> /dev/null 
try 3
echo

echo "===> Testing stopping a container of the app service"
tutum container stop --sync "$APPNAME-3" >> /dev/null
try 2
echo

echo "===> Testing starting a container of the app service"
tutum container start --sync "$APPNAME-3" >> /dev/null
try 3 
echo

echo "===> Testing terminating a container of the app service"
tutum container terminate --sync "$APPNAME-3" >> /dev/null
try 2
echo

echo "===> Testing zero-downtime redeployment"

tryZeroDown &
ZERO_PID=$!
echo "======> Load Balancer reload"
tutum service redeploy --sync $LOADNAME >> /dev/null 
kill -15 $ZERO_PID 
sleep 1
echo

tryZeroDown &
ZERO_PID=$!
echo "======> App service reload"
tutum service redeploy --sync $APPNAME >> /dev/null
kill -15 $ZERO_PID
sleep 1
echo

tryZeroDown &
ZERO_PID=$!
echo "======> Reload the whole stack"
tutum stack redeploy --sync $STACKNAME >> /dev/null 
kill -15 $ZERO_PID
sleep 1
echo

#TODO Add testcases for the BACKEND_PORT(S) envvars

echo "===> Sanity check to make sure there are 2 containers remaining after the tests"
try 2 
echo

quit
