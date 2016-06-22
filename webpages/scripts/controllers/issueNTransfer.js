// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

'use strict';

/**
 * @ngdoc function
 * @name bitmarkWebguiApp.controller:IssueNTransferCtrl
 * @description
 * # IssueNTransferCtrl
 * Controller of the bitmarkWebguiApp
 */
angular.module('bitmarkWebguiApp')
    .controller('IssueNTransferCtrl', function ($scope, $interval, $location, httpService, configuration, BitmarkCliConfig, BitmarkPayConfig) {
        if(configuration.getConfiguration().bitmarkCliConfigFile.length == 0){
            $location.path('/login');
        }

        if($location.path() == "/issue" ){
            $scope.showIssueView = true;
        }else if($location.path() == "/transfer"){
            $scope.showIssueView = false;
        }else {
            $location.path('/main');
        }

        var chain = configuration.getConfiguration().chain;
        var bitmarkCliConfigFile = BitmarkCliConfig[chain];
        var bitmarkPayConfigFile = BitmarkPayConfig[chain];;

        $scope.init = function(){
            localInit(chain);
        };


        var localInit = function(bitmarkChain){
            // get config file by chan type
            $scope.showWaiting = false;

            // default setup config
            $scope.bitmarkCliInfoSuccess = false;

            // default info config
            $scope.infoErr = {
                show: false,
                msg: ""
            };
            $scope.infoAlert = {
                show: false,
                msg: ""
            };

            // default issue config
            $scope.issueConfig = {
                network:  chain,
                pay_config: bitmarkPayConfigFile,
                identity:"",
                asset:"",
                description:"",
                fingerprint:"",
                quantity:1,
                password:""
            };

            // transfer config
            $scope.transferConfig = {
                network:  chain,
                pay_config: bitmarkPayConfigFile,
                identity:"",
                txid:"",
                receiver:"",
                password:""
            };

            getInfo();
        };

        var infoPromise;
        var infoJobHash = "";
        var infoWaitingTime = 10; // 10s
        var pollInfoCount = 0;
        var getBitmarkPayInfoInterval = function(){
            return $interval(function(){
                httpService.send("getBitmarkPayStatus", {
                    job_hash: infoJobHash
                }).then(
                    function(statusResult){
                        switch(statusResult){
                        case "success":
                            $interval.cancel(infoPromise);
                            pollInfoCount = 0;
                            $scope.showWaiting = false;
                            httpService.send("getBitmarkPayResult", {"job_hash":infoJobHash}).then(function(payResult){
                                $scope.onestepStatusResult.pay_result = payResult;
                                $scope.bitmarkCliInfoSuccess = true;
                            },function(payErr){
                                $scope.infoErr.msg = payErr;
                                $scope.infoErr.show = true;
                            });
                            break;
                        case "running":
                            pollInfoCount++;
                            if(pollInfoCount*3 > infoWaitingTime && !$scope.infoAlert.show){
                                $scope.infoAlert.msg = "The bitmark-pay seems running for a long time, please check your bitcoin and bitmark-pay configuration. Would you like to stop the process?";
                                $scope.showWaiting = false;
                                $scope.infoAlert.show = true;
                            }
                            break;
                        case "fail":
                            $interval.cancel(infoPromise);
                            $scope.infoErr.msg = "bitmark-pay error: "+statusResult;
                            $scope.infoErr.show = true;
                            $scope.showWaiting = false;
                            break;
                        }
                    });
            }, 3*1000);
        };
        var getInfo = function(){
            $scope.showWaiting = true;
            $scope.infoErr.show = false;
            $scope.infoAlert.show = false;

            httpService.send("onestepStatus",{
                network: chain,
                pay_config: bitmarkPayConfigFile
            }).then(function(infoResult){
                $interval.cancel(infoPromise);
                infoJobHash = infoResult.job_hash;
                $scope.bitmarkCliInfoSuccess = false;
                $scope.onestepStatusResult = infoResult;
                infoPromise = getBitmarkPayInfoInterval();

            }, function(infoErr){
                if( infoErr == "Failed to get bitmark-cli info") {
                    // bitmark-cli never setup, show setup view
                    $scope.showWaiting = false;
                } else {
                    httpService.send('getBitmarkPayJob').then(function(jobHash){
                        infoJobHash = jobHash;
                        $scope.showWaiting = false;
                        if(jobHash != "") {
                            // bitmark-pay error
                            infoPromise = getBitmarkPayInfoInterval();
                            $scope.infoAlert.msg = "The previous bitmark-pay is running. Would you like to stop the process?";
                            $scope.infoAlert.show = true;
                        } else {
                            $scope.infoErr.msg = infoErr;
                            $scope.infoErr.show = true;
                        }
                    });
                }
            });
        };

        $scope.clearErrAlert = function(type) {
            switch(type) {
            case "issue":
                $scope.issueResult = null;
            case "transfer":
                $scope.transferResult = null;
            }
        };

        var issuePromise;
        $scope.submitIssue = function(){
            $scope.clearErrAlert('issue');
            $scope.issueResult = {
                type:"danger",
                msg: "",
                failStart: null,
                cliResult: null
            };
            $scope.issueConfig.identity = $scope.onestepStatusResult.cli_result.identities[0].name;
            httpService.send("onestepIssue", $scope.issueConfig).then(
                function(result){
                    issuePromise = $interval(function(){
                        httpService.send("getBitmarkPayStatus", {
                            job_hash: result.job_hash
                        }).then(function(payStatus){
                            if(payStatus == "success"){
                                $interval.cancel(issuePromise);
                                httpService.send("getBitmarkPayResult", {
                                    "job_hash": result.job_hash
                                }).then(function(payResult){
                                    $scope.issueResult.type = "success";
                                    $scope.issueResult.msg = "Pay success!";
                                    $scope.issueResult.cliResult = result.cli_result;
                                }, function(payErr){
                                    $scope.issueResult.type = "danger";
                                    if(payErr.cli_result != null) {
                                        $scope.issueResult.msg = "Pay failed";
                                        $scope.issueResult.failStart = payErr.fail_start;
                                        $scope.issueResult.cliResult = payErr.cli_result;
                                    } else{
                                        $scope.issueResult.msg = payErr;
                                    }
                                });
                            }else{
                            // TODO: see if bitmark-pay is still running
                            }
                        });
                    }, 3*1000);
                },
                function(errResult){
                    $scope.issueResult.type = "danger";
                    if(errResult.cli_result != null) {
                        $scope.issueResult.msg = "Pay failed";
                        $scope.issueResult.failStart = errResult.fail_start;
                        $scope.issueResult.cliResult = errResult.cli_result;
                    } else{
                        $scope.issueResult.msg = errResult;
                    }
                });
        };


        var transferPromise;
        $scope.submitTransfer = function(){
            $scope.clearErrAlert('transfer');
            $scope.transferResult = {
                type:"danger",
                msg: "",
                cliResult: null
            };
            $scope.transferConfig.identity = $scope.onestepStatusResult.cli_result.identities[0].name;

            httpService.send("onestepTransfer", $scope.transferConfig).then(
                function(result){
                    transferPromise = $interval(function(){
                        httpService.send("getBitmarkPayStatus", {
                            job_hash: result.job_hash
                        }).then(function(payStatus){
                            if(payStatus == "success"){
                                $interval.cancel(transferPromise);
                                httpService.send("getBitmarkPayResult", {"job_hash": result.job_hash}).then(function(payResult){
                                    $scope.transferResult.type = "success";
                                    $scope.transferResult.msg = "Pay success!";
                                    $scope.transferResult.cliResult = result.cli_result;
                                },function(payErr){
                                    // TODO: pay error
                                });
                            } else {
                                // TODO: see if bitmark-pay is still running
                            }
                        });
                    }, 3*1000);


                }, function(errResult){
                    $scope.transferResult.type = "danger";
                    if(errResult.cli_result != null) {
                        $scope.transferResult.msg = "Pay failed";
                        $scope.transferResult.cliResult = errResult.cli_result;
                    } else{
                        $scope.transferResult.msg = errResult;
                    }
                });
        };

        var killPromise;
        var killBitmarkPayStatusProcess = function(jobHash, alertObj){
            $scope.showWaiting = true;
            httpService.send('stopBitmarkPayProcess', {"job_hash": jobHash}).then(function(result){
                $interval.cancel(killPromise);
                killPromise = $interval(function(){
                    httpService.send("getBitmarkPayStatus", {
                        job_hash: jobHash
                    }).then(function(payStatus){
                        if(payStatus == "stopped"){
                            $interval.cancel(killPromise);
                            $scope.showWaiting = false;
                            alertObj.show = false;
                        }
                    });
                }, 3*1000);
            }, function(err){
                alertObj.show = true;
                alertObj.msg = err;
                $scope.showWaiting = false;
            });
        };

        $scope.killPayProcess = function(type, kill){
            switch(type){
            case "setup":
                $interval.cancel(setupPromise);
                pollSetupCount = 0;
                killBitmarkPayStatusProcess(setupJobHash, $scope.setupAlert);
                break;
            case "info":
                if(kill){
                    $interval.cancel(infoPromise);
                    pollInfoCount = 0;
                    if(infoJobHash == "" || infoJobHash == null) {
                        httpService.send('getBitmarkPayJob').then(function(jobHash){
                            infoJobHash = jobHash;
                            killBitmarkPayStatusProcess(infoJobHash, $scope.infoAlert);
                        });
                    }else{
                        killBitmarkPayStatusProcess(infoJobHash, $scope.infoAlert);
                    }
                }else{
                    $scope.infoAlert.show = false;
                    $scope.showWaiting = true;
                    pollInfoCount = 0;
                }
                break;
            }
        };

        $scope.$on("$destroy", function(){
            $interval.cancel(infoPromise);
            $interval.cancel(issuePromise);
            $interval.cancel(transferPromise);
            $interval.cancel(killPromise);
        });


  });
