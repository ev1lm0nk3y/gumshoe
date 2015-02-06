(function() {
	var app = angular.module('gumshoe', ['fundoo.services']);

  app.controller('ShowController', ['$log', '$http', function($log, $http){
    var showCtrl = this;
    showCtrl.shows = [];
    showCtrl.newShow = {};
    var showAddForm = false;

    $http.get("/api/shows").success(function(data){
      showCtrl.shows = data.Shows;
    });

    this.boolConv = function(str) {
      switch(str) {
        case "true":
          return true;
        case "false":
          return false;
      }
    };

    this.addShow = function() {
      $log.log("addShow");
      this.newShow.episodal = this.boolConv(this.newShow.episodal);
      $http.post("/api/show/new", this.newShow).success(function(data){
        showCtrl.shows.push(showCtrl.newShow);
        showCtrl.newShow = {};
        showCtrl.showAddForm = false;
      }).error(function(data, status, headers, config){
        $log.log(data, status, headers, config);
      });
    };

    this.showEditForm = function(index) {
      if(typeof showCtrl.shows[index].showEditForm == "undefined") {
        showCtrl.shows[index].showEditForm = true;
      } else {
        showCtrl.shows[index].showEditForm = !showCtrl.shows[index].showEditForm;
      }
    };

    this.isEditFormVisible = function(index) {
      if(typeof showCtrl.shows[index].showEditForm == "undefined") {
        showCtrl.shows[index].showEditForm = false;
      }
      return showCtrl.shows[index].showEditForm;
    };

    this.editShow = function(index) {
      newShow = this.shows[index];
      newShow.episodal = this.boolConv(newShow.episodal);
      $http.post("/api/show/update/" + newShow.ID, newShow).success(function(){
        showCtrl.showEditForm(index);
      }).error(function(data, status, headers, config){
        $log.log(data, status, headers, config);
      });
    };

    this.deleteShow = function(index) {
      title = showCtrl.shows[index].title;
      if(window.confirm("Delete " + title + "?")) {
        $http.delete("/api/show/delete/" + showCtrl.shows[index].ID).success(function(data){
          showCtrl.shows.splice(index, 1);
        });
      };
    };

  } ] );

  app.directive("gumshoeTabs", function() {
     return {
       restrict: "E",
       templateUrl: "gumshoe-tabs.html",
       controller: function() {
         this.tab = 1;

         this.isSet = function(checkTab) {
           return this.tab === checkTab;
         };

         this.setTab = function(activeTab) {
           this.tab = activeTab;
         };
       },
       controllerAs: "tab"
     };
   });

	app.directive("gumshoeStatus", function() {
		return {
      restrict: 'E',
      templateUrl: "gumshoe-status.html"
    };
  });
	app.directive("gumshoeShows", function() {
		return {
      restrict: 'E',
      templateUrl: "gumshoe-shows.html"
    };
  });
	app.directive("gumshoeQueue", function() {
		return {
      restrict: 'E',
      templateUrl: "gumshoe-queue.html"
    };
  });

  app.controller('SettingsDialog', ['$scope', 'createDialog', function($scope, createDialogService) {
    $scope.launchSettingsModal = function() {
      createDialogService({
        id: 'gumshoeSettings',
        template: 'gumshoe-settings.html',
        title: 'Gumshoe Settings',
        backdrop: true,
        success: {label: 'Save Changes', fn: this.saveSettings()},
        controller: 'SettingsController'
      }, {
        settings: $http({
          method: 'JSONP',
          url: '/api/settings/'})
      });
    };
    $scope.launchHelpModal = function() {console.log('show help message.');};
  }]);

  app.controller('SettingsController', ['$scope', 'SettingsFactory', 'settings',
      function($scope, SettingsFactory, settings) {
        $scope.settings = settings;
  }]);

  app.factory('SettingsFactory', function() {
    return {
      sample: function() {
        console.log('Sample');
      }
    };
  });


})();
