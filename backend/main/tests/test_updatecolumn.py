from rest_framework.test import APITestCase
from ..models import Column, Board, Team, Task
from ..util import new_admin


class UpdateColumns(APITestCase):
    endpoint = '/columns/?id='

    def setUp(self):
        team = Team.objects.create()
        board = Board.objects.create(name='My Board', team=team)
        self.column = Column.objects.create(order=0, board=board)
        self.tasks = [
            Task.objects.create(
                id=i,
                title=str(i),
                order=i,
                column=self.column
            ) for i in range(0, 5)
        ]
        self.admin = new_admin(team)

    def test_order_success(self):
        request_data = list(map(
            lambda task: {
                'id': task.id,
                'title': task.title,
                'order': 5 - task.order
            },
            self.tasks
        ))
        response = self.client.patch(f'{self.endpoint}${self.column.id}',
                                     request_data,
                                     format='json',
                                     HTTP_AUTH_USER=self.admin['username'],
                                     HTTP_AUTH_TOKEN=self.admin['token'])
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.data, {
            'msg': 'Column and all its tasks updated successfully.',
            'id': self.column.id,
        })
        new_tasks = Task.objects.get(column_id=self.column.id)
        map(
            lambda task: self.assertEqual(task.order, 5 - int(task.title)),
            new_tasks
        )
