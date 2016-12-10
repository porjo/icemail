var apiURL = '//localhost:8080/api';

var	msgLimit = 35;

var fields = [
	"From",
	"To",
	"Subject"
];

$(function() {
	var app = new Vue({
		el: '#app',
		data: {
			result: {
				emails: [],
				total: 0,
				offset: 0
			},
			fields: fields,
			searchDays: 0,
			showModal: false,
			request: {
				query: "",
				limit: msgLimit,
				offset: 0,
				locations: fields,
			},
			modal: {
				Title: '',
				Body: '',
				ID: 0
			}
		},

		created: function () {
			this.searchMsg()
		},

		methods: {

			closeModal: function() {
				$("body").removeClass('modal-open');
				this.showModal = false;
			},

			sendMsg: function(id) {
				var self = this;
				$.get(apiURL + '/mail/' + id, function(data) {
					console.log(data);
				});
			},

			viewMsg: function(id) {
				var haveSel = getSelection().toString().length > 0;
				if( haveSel ) {
					return;
				}
				var self = this;
				$("body").addClass('modal-open');
				$.get(apiURL + '/search/' + id, function(data) {
					self.modal.Title = data.Emails[0].Header.Subject[0];
					self.modal.Body = data.Emails[0].Body;
					self.modal.ID = data.Emails[0].ID;
					self.showModal = true;
				});
			},

			toggleSearchOptions: function() {
				$('#search_options').slideToggle();
			},

			searchMsg: function(direction) {
				var self = this;

				var request = $.extend({}, this.request);

				if( this.searchDays > 0) {
					var startTime = new Date();
					startTime.setDate(startTime.getDate() - this.searchDays);
					request.starttime = startTime.toISOString();
				}

				if( typeof direction !== "undefined" ) {
					var offset = 0;
					if(direction == 'back') {
						offset = this.result.offset - request.limit;
					} else {
						offset = this.result.offset + request.limit;
					}
					if( offset < 0  || offset > this.result.total ) {
							// abort search out of limits
							return;
					} else {
						request.offset = offset;
					}
				}

				$.ajax({
					url: apiURL + '/search',
					type: 'post',
					data: JSON.stringify(request),
					contentType: 'application/json',
					dataType: 'json',
					success: function(data) {
						cleanData(data);
						self.result.emails = data.Emails;
						self.result.total = data.Total;
						self.result.offset = data.Offset;
					}
				});
			}
		}
	});

	Vue.component('modal', {
		template: '#modal-template'
	})

	function cleanData(data) {
		$.each(data.Emails, function(k,v) {
			if( typeof v.Header.Date === 'undefined' )
				v.Header.Date = '';
			if( typeof v.Header.From === 'undefined' )
				v.Header.From = '';
			if( typeof v.Header.To === 'undefined' )
				v.Header.To = '';
		});
	}
});
