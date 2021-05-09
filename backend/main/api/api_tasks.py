from rest_framework.decorators import api_view
from rest_framework.response import Response
from rest_framework.exceptions import ErrorDetail
from ..models import Task, Column
from ..serializers.ser_task import TaskSerializer
from ..serializers.subtaskserializer import SubtaskSerializer
from ..validation.val_auth import \
    authenticate, authorize, not_authenticated_response
from ..validation.val_column import validate_column_id
from ..validation.val_task import validate_task_id


@api_view(['POST', 'PATCH', 'DELETE'])
def tasks(request):
    username = request.META.get('HTTP_AUTH_USER')
    token = request.META.get('HTTP_AUTH_TOKEN')

    user, authentication_response = authenticate(username, token)
    if authentication_response:
        return authentication_response

    if request.method == 'POST':
        authorization_response = authorize(username)
        if authorization_response:
            return authorization_response

        column_id = request.data.get('column')
        validation_response = validate_column_id(column_id)
        if validation_response:
            return validation_response

        try:
            column = Column.objects.select_related(
                'board'
            ).prefetch_related(
                'task_set'
            ).get(id=column_id)
        except Column.DoesNotExist:
            return Response({
                'column_id': ErrorDetail(string='Column not found.',
                                         code='not_found')
            }, 404)

        if column.board.team_id != user.team_id:
            return not_authenticated_response

        for task in column.task_set.all():
            task.order += 1

        Task.objects.bulk_update(column.task_set.all(), ['order'])

        # save task
        task_serializer = TaskSerializer(
            data={'title': request.data.get('title'),
                  'description': request.data.get('description'),
                  'order': 0,
                  'column': request.data.get('column')}
        )
        if not task_serializer.is_valid():
            return Response(task_serializer.errors, 400)
        task = task_serializer.save()

        # save subtasks
        subtasks = request.data.get('subtasks')
        subtasks_data = [
            {
                'title': subtask,
                'order': i,
                'task': task.id
            } for i, subtask in enumerate(subtasks)
        ] if subtasks else []
        subtask_serializer = SubtaskSerializer(data=subtasks_data, many=True)
        if not subtask_serializer.is_valid():
            task.delete()
            return Response({
                'subtask': subtask_serializer.errors
            }, 400)
        subtask_serializer.save()

        return Response({
            'msg': 'Task creation successful.',
            'task_id': task.id
        }, 201)

    if request.method == 'PATCH':
        authorization_response = authorize(username)
        if authorization_response:
            return authorization_response

        task_id = request.query_params.get('id')
        validation_response = validate_task_id(task_id)
        if validation_response:
            return validation_response

        task = Task.objects.select_related(
            'column',
            'column__board'
        ).prefetch_related(
            'subtask_set'
        ).get(id=task_id)

        if task.column.board.team_id != user.team_id:
            return not_authenticated_response

        if 'title' in request.data.keys() and not request.data.get('title'):
            return Response({
                'title': ErrorDetail(string='Task title cannot be empty.',
                                     code='blank')
            }, 400)

        order = request.data.get('order')
        if 'order' in request.data.keys() and (order == '' or order is None):
            return Response({
                'order': ErrorDetail(string='Task order cannot be empty.',
                                     code='blank')
            }, 400)

        if 'column' in request.data.keys():
            column_id = request.data.get('column')
            validation_response = validate_column_id(column_id)
            if validation_response:
                return validation_response

        subtasks = request.data.pop(
            'subtasks'
        ) if 'subtasks' in request.data.keys() else None

        # update tasks
        task_serializer = TaskSerializer(task, data=request.data, partial=True)
        if not task_serializer.is_valid():
            return Response(task_serializer.errors, 400)
        task = task_serializer.save()

        # update subtasks
        if subtasks:
            task.subtask_set.all().delete()
            subtasks_data = [
                {
                    'title': subtask['title'],
                    'order': subtask['order'],
                    'task': task.id,
                    'done': subtask['done']
                } for subtask in subtasks
            ]
            subtask_serializer = SubtaskSerializer(data=subtasks_data,
                                                   many=True)
            if not subtask_serializer.is_valid():
                return Response({'subtasks': subtask_serializer.errors}, 400)
            subtask_serializer.save()

        return Response({
            'msg': 'Task update successful.',
            'id': task.id
        }, 200)

    if request.method == 'DELETE':
        authorization_response = authorize(username)
        if authorization_response:
            return authorization_response

        task_id = request.query_params.get('id')

        validation_response = validate_task_id(task_id)
        if validation_response:
            return validation_response

        try:
            task = Task.objects.select_related(
                'column',
                'column__board',
            ).get(id=task_id)
        except Task.DoesNotExist:
            return Response({
                'task_id': ErrorDetail(string='Task not found.',
                                       code='not_found')
            }, 404)

        if task.column.board.team_id != user.team_id:
            return not_authenticated_response

        task.delete()

        return Response({
            'msg': 'Task deleted successfully.',
            'id': task_id,
        })
